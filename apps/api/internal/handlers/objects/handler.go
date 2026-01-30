package objects

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/services"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
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
}

func (h *ObjectHandler) GetTableHash(c *gin.Context) {
	tenantId := c.GetString("tenantId")
	var req GetTableHashRequest
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
	if err := c.BindJSON(&enclaveReqBody); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session init request: %v", err)})
		return
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
	resp := GetTableHashResponse{
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
	var req CreateObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tableHash := c.Param("table-hash")
	// Get tenant ID from context
	tenantId := c.GetString("tenantId")

	// Generate object ID and S3 key
	objectId := uuid.NewString()
	s3Key := generateS3Key(tenantId, objectId, 1)

	obj := models.NewObject(tenantId, tableHash, objectId, s3Key, req.KMSEncryptedDEK, req.MasterEncryptedDEK, req.DEKNonce)
	if req.Sensitivity != "" {
		obj.Sensitivity = req.Sensitivity
	} else {
		obj.Sensitivity = models.SensitivityLow
	}
	obj.Status = models.StatusPending
	obj.TTL = time.Now().Add(h.PresignTTL).Unix()

	uploadUrl, err := h.S3Client.PresignPutObjectUrl(ctx, s3Key, h.PresignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned URL: " + err.Error()})
		return
	}

	if err := h.Dynamo.CreateObjectWithIndexes(ctx, tableHash, obj, req.Indexes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create object in DynamoDB: " + err.Error()})
		return
	}

	resp := CreateObjectResponse{
		ObjectID:  objectId,
		UploadURL: uploadUrl,
		ExpiresIn: int64(h.PresignTTL.Seconds()),
		CreatedAt: obj.CreatedAt,
	}

	c.JSON(http.StatusCreated, resp)
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

	getUrl, err := h.S3Client.PresignGetObjectUrl(ctx, obj.S3Key, h.PresignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned GET URL: " + err.Error()})
		return
	}

	resp := GetObjectResponse{
		ObjectID:  id,
		GetURL:    getUrl,
		CreatedAt: obj.CreatedAt,
		UpdatedAt: obj.UpdatedAt,
		Version:   obj.Version,
	}

	c.JSON(http.StatusOK, resp)
}

// Helper function to generate S3 key
func generateS3Key(tenantId string, objectId string, version int32) string {
	return "tenant-" + tenantId + "/objects/" + objectId + "/v" + fmt.Sprintf("%d", version)
}
