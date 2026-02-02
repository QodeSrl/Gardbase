package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/services"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/api/objects"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ObjectHandler struct {
	Vsock      *services.Vsock
	S3Client   *storage.S3Client
	Dynamo     *storage.DynamoClient
	KMS        *services.KMS
	PresignTTL time.Duration
	BaseURL    string
}

func (h *ObjectHandler) GetTableHash(c *gin.Context) {
	tenantId := c.GetString("tenantId")
	var req objects.GetTableHashRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenant, err := h.Dynamo.GetTenant(c.Request.Context(), tenantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant from DynamoDB: " + err.Error()})
		return
	}
	attDoc, err := h.Vsock.RequestAttestationDocument()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get attestation document: " + err.Error()})
		return
	}
	wrappedTableSalt, err := base64.StdEncoding.DecodeString(tenant.WrappedTableSalt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode table salt: " + err.Error()})
		return
	}
	// Decrypt table salt using KMS
	tableSalt, err := h.KMS.Decrypt(c.Request.Context(), wrappedTableSalt, attDoc, tenantId, services.PurposeTableSalt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt table salt: " + err.Error()})
		return
	}
	// Build enclave request
	enclaveReqBody := enclaveproto.SessionGenerateTableHashRequest{
		SessionID:                 req.SessionID,
		SessionEncryptedTableName: req.SessionEncryptedTableName,
		SessionTableNameNonce:     req.SessionTableNameNonce,
		TableSalt:                 base64.StdEncoding.EncodeToString(tableSalt.CiphertextForRecipient),
	}
	payloadBytes, err := json.Marshal(enclaveReqBody)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session init request: %v", err)})
		return
	}
	enclaveReq := enclaveproto.Request{
		Type:    "session_generate_table_hash",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := h.Vsock.SendToEnclave(enclaveReq, 10*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Generate table hash from tenant table salt
	var res enclaveproto.Response[enclaveproto.SessionGenerateTableHashResponse]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session init response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	resp := objects.GetTableHashResponse{
		TableHash: res.Data.TableHash,
	}
	c.JSON(http.StatusOK, resp)
}

/*
The Create method handles the creation of a new object. It expects a JSON payload with tenant ID, encrypted DEK, optional sensitivity, and indexes.
It generates a unique object ID and S3 key, sets the object's status to pending, and calculates a TTL.
It then generates a presigned URL for uploading the object to S3.
The object metadata and indexes are stored in DynamoDB using the CreateObjectWithIndexes method.
Finally, it responds with the object ID, S3 key, upload URL, and expiration time.
*/
func (h *ObjectHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	var req objects.CreateObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tableHash := c.Param("table-hash")
	// Get tenant ID from context
	tenantId := c.GetString("tenantId")

	// Generate object ID
	objectId := uuid.NewString()

	obj := models.NewObject(tenantId, tableHash, objectId, req.KMSEncryptedDEK, req.MasterEncryptedDEK, req.DEKNonce)
	var uploadUrl string

	if req.BlobSize > 100*1024 {
		// If blob size > 100KB, use S3 for storage and generate presigned URL
		s3Key := generateS3Key(tenantId, tableHash, objectId, 1)
		obj.S3Key = s3Key
		presignedUrl, err := h.S3Client.PresignPutObjectUrl(ctx, s3Key, h.PresignTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned URL: " + err.Error()})
			return
		}
		uploadUrl = presignedUrl
	} else {
		// If blob size <= 100KB, use inline storage in DynamoDB
		obj.EncryptedBlob = "" // Will be filled later
		uploadUrl = fmt.Sprintf("%s/objects/%s/%s/upload-inline", h.BaseURL, tableHash, objectId)
	}

	if req.Sensitivity != "" {
		obj.Sensitivity = req.Sensitivity
	} else {
		obj.Sensitivity = models.SensitivityLow
	}
	obj.Status = models.StatusPending
	obj.TTL = time.Now().Add(h.PresignTTL).Unix()

	if err := h.Dynamo.CreateObjectWithIndexes(ctx, tableHash, obj, req.Indexes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create object in DynamoDB: " + err.Error()})
		return
	}

	resp := objects.CreateObjectResponse{
		ObjectID:  objectId,
		UploadURL: uploadUrl,
		ExpiresIn: int64(h.PresignTTL.Seconds()),
		CreatedAt: obj.CreatedAt,
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ObjectHandler) UploadInline(c *gin.Context) {
	ctx := c.Request.Context()
	tenantId := c.GetString("tenantId")
	id := c.Param("id")
	tableHash := c.Param("table-hash")

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body: " + err.Error()})
		return
	}
	if len(body) > 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Inline blob size exceeds 100KB limit"})
		return
	}

	obj, err := h.Dynamo.GetObject(ctx, tenantId, tableHash, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get object from DynamoDB: " + err.Error()})
		return
	}
	if obj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}
	if obj.S3Key != "" || obj.EncryptedBlob != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Object is not eligible for inline upload"})
		return
	}

	if err := h.Dynamo.UpdateObjectInlineBlob(ctx, tenantId, tableHash, id, base64.StdEncoding.EncodeToString(body)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update object inline blob in DynamoDB: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Inline blob uploaded successfully"})
}

/*
The Get method handles the retrieval of an object through its ID.
*/
func (h *ObjectHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	tenantId := c.GetString("tenantId")
	id := c.Param("id")
	tableHash := c.Param("table-hash")

	obj, err := h.Dynamo.GetObject(ctx, tenantId, tableHash, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get object from DynamoDB: " + err.Error()})
		return
	}
	if obj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}
	if obj.Status != models.StatusReady {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Object is not in READY status"})
		return
	}

	getUrl := ""
	encryptedBlob := obj.EncryptedBlob

	if obj.S3Key != "" {
		getUrl, err = h.S3Client.PresignGetObjectUrl(ctx, obj.S3Key, h.PresignTTL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned GET URL: " + err.Error()})
			return
		}
	}

	resp := objects.GetObjectResponse{
		ObjectID:         id,
		GetURL:           getUrl,
		EncryptedBlob:    encryptedBlob,
		KMSWrappedDEK:    obj.KMSWrappedDEK,
		MasterWrappedDEK: obj.MasterWrappedDEK,
		DEKNonce:         obj.DEKNonce,
		CreatedAt:        obj.CreatedAt,
		UpdatedAt:        obj.UpdatedAt,
		Version:          obj.Version,
	}

	c.JSON(http.StatusOK, resp)
}

// Helper function to generate S3 key
func generateS3Key(tenantId string, tableHash string, objectId string, version int32) string {
	return "tenant-" + tenantId + "/" + tableHash + "/" + objectId + "/v" + fmt.Sprintf("%d", version)
}
