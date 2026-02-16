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

/*
The GetTableHash method handles the generation of a table hash for a session. It expects a JSON payload with tenant ID, session ID, optional encrypted table name, and optional table name nonce.
It retrieves the tenant information from DynamoDB and decrypts the table salt using KMS. It then builds a request to the enclave to generate the table hash using the session information and decrypted table salt.
Finally, it responds with the generated table hash or an error if any step fails.
*/
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

func (h *ObjectHandler) RequestPutLarge(c *gin.Context) {
	ctx := c.Request.Context()
	var req objects.RequestPutLargeObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Get tenant ID from context
	tenantId := c.GetString("tenantId")

	// ObjectID and Version must be consistent
	if req.ObjectID != "" && req.Version == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Version must be provided for updates"})
		return
	}
	if req.ObjectID == "" && req.Version != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Version should not be provided for new objects"})
		return
	}

	if req.BlobSize <= 100*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Blob size must be greater than 100KB for large object upload"})
		return
	}

	var objectId string
	var expectedVersion int32

	if req.ObjectID == "" {
		// Create
		objectId = uuid.NewString()
		expectedVersion = 1
	} else {
		objectId = req.ObjectID
		// Update - check if object exists
		obj, err := h.Dynamo.GetObject(ctx, tenantId, req.TableHash, req.ObjectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get object from DynamoDB: " + err.Error()})
			return
		}
		if obj == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
			return
		}
		if obj.Status == models.StatusDeleted {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot update a deleted object"})
			return
		}
		if obj.Version != req.Version-1 {
			c.JSON(http.StatusConflict, gin.H{"error": "Version mismatch. Current version is " + fmt.Sprintf("%d", obj.Version)})
			return
		}
		expectedVersion = obj.Version + 1
	}

	s3Key := generateS3Key(tenantId, req.TableHash, objectId, expectedVersion)

	uploadUrl, err := h.S3Client.PresignPutObjectUrl(ctx, s3Key, h.PresignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate presigned PUT URL: " + err.Error()})
		return
	}

	resp := objects.RequestPutLargeObjectResponse{
		ObjectID:        objectId,
		UploadURL:       uploadUrl,
		ExpectedVersion: expectedVersion,
		ExpiresIn:       int64(h.PresignTTL.Seconds()),
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ObjectHandler) ConfirmPutLarge(c *gin.Context) {
	ctx := c.Request.Context()
	var req objects.ConfirmPutLargeObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Get tenant ID from context
	tenantId := c.GetString("tenantId")

	// Verify uploaded object exists in S3
	s3Key := generateS3Key(tenantId, req.TableHash, req.ObjectID, req.Version)
	exists, err := h.S3Client.CheckObjectExists(ctx, s3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check object in S3: " + err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uploaded object not found in S3"})
		return
	}

	now := time.Now().UTC()

	if req.Version == 1 {
		// Create
		obj := models.NewObject(tenantId, req.TableHash, req.ObjectID, req.KMSEncryptedDEK, req.MasterEncryptedDEK, req.DEKNonce)
		obj.S3Key = s3Key
		obj.Version = 1
		obj.Status = models.StatusReady
		obj.CreatedAt = now
		obj.UpdatedAt = now

		if err := h.Dynamo.CreateObjectWithIndexes(ctx, req.TableHash, obj, req.Indexes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to put object in DynamoDB: " + err.Error()})
			return
		}

		resp := objects.ConfirmPutLargeObjectResponse{
			ObjectID:  req.ObjectID,
			CreatedAt: obj.CreatedAt,
			UpdatedAt: obj.UpdatedAt,
			TableHash: req.TableHash,
			Version:   obj.Version,
		}
		c.JSON(http.StatusOK, resp)
		return
	}

	// Update
	obj, err := h.Dynamo.UpdateObjectWithIndexes(ctx, tenantId, req.TableHash, req.ObjectID, req.Version-1, func(obj *models.Object) {
		obj.S3Key = s3Key
		obj.KMSWrappedDEK = req.KMSEncryptedDEK
		obj.MasterWrappedDEK = req.MasterEncryptedDEK
		obj.DEKNonce = req.DEKNonce
		obj.UpdatedAt = now
		obj.Version = req.Version
		if req.Sensitivity != "" {
			obj.Sensitivity = req.Sensitivity
		}
	}, req.Indexes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update object in DynamoDB: " + err.Error()})
		return
	}

	resp := objects.ConfirmPutLargeObjectResponse{
		ObjectID:  req.ObjectID,
		CreatedAt: obj.CreatedAt,
		UpdatedAt: obj.UpdatedAt,
		TableHash: req.TableHash,
		Version:   req.Version,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ObjectHandler) Put(c *gin.Context) {
	ctx := c.Request.Context()
	var req objects.PutObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate: version and ID must be consistent
	if req.ObjectID != "" && req.Version == 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Version must be provided for updates"})
		return
	}
	if req.ObjectID == "" && req.Version != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Version should not be provided for new objects"})
		return
	}

	// Get tenant ID from context
	tenantId := c.GetString("tenantId")
	now := time.Now().UTC()

	if req.ObjectID == "" {
		// Create
		objectId := uuid.NewString()
		obj := models.NewObject(tenantId, req.TableHash, objectId, req.KMSEncryptedDEK, req.MasterEncryptedDEK, req.DEKNonce)
		obj.EncryptedBlob = req.EncryptedBlob
		obj.Version = 1
		obj.CreatedAt = now
		obj.UpdatedAt = now
		obj.Status = models.StatusReady
		if req.Sensitivity != "" {
			obj.Sensitivity = req.Sensitivity
		} else {
			obj.Sensitivity = models.SensitivityLow
		}
		if err := h.Dynamo.CreateObjectWithIndexes(ctx, req.TableHash, obj, req.Indexes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create object in DynamoDB: " + err.Error()})
			return
		}

		resp := objects.PutObjectResponse{
			ObjectID:  objectId,
			CreatedAt: obj.CreatedAt,
			UpdatedAt: obj.UpdatedAt,
			TableHash: req.TableHash,
			Version:   obj.Version,
		}

		c.JSON(http.StatusCreated, resp)
		return
	}

	// Update
	obj, err := h.Dynamo.UpdateObjectWithIndexes(ctx, tenantId, req.TableHash, req.ObjectID, req.Version-1, func(obj *models.Object) {
		obj.EncryptedBlob = req.EncryptedBlob
		obj.S3Key = "" // clear s3key if switching from large object to inline
		obj.KMSWrappedDEK = req.KMSEncryptedDEK
		obj.MasterWrappedDEK = req.MasterEncryptedDEK
		obj.DEKNonce = req.DEKNonce
		obj.UpdatedAt = now
		obj.Version = req.Version
		if req.Sensitivity != "" {
			obj.Sensitivity = req.Sensitivity
		}
	}, req.Indexes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update object in DynamoDB: " + err.Error()})
		return
	}

	resp := objects.PutObjectResponse{
		ObjectID:  req.ObjectID,
		CreatedAt: obj.CreatedAt,
		UpdatedAt: obj.UpdatedAt,
		TableHash: req.TableHash,
		Version:   obj.Version,
	}

	c.JSON(http.StatusCreated, resp)
}

/*
The Get method handles the retrieval of an object through its ID.
It expects a JSON payload with tenant ID, table hash, and object ID. It retrieves the object from DynamoDB using the GetObject method.
If the object is found and is in READY status, it generates a presigned GET URL if the object is stored in S3.
Finally, it responds with the object ID, presigned GET URL (if applicable), encrypted blob (if inline), KMS-wrapped DEK, master-wrapped DEK, DEK nonce, and timestamps.
*/
func (h *ObjectHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	tenantId := c.GetString("tenantId")
	var req objects.GetObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	obj, err := h.Dynamo.GetObject(ctx, tenantId, req.TableHash, req.ObjectID)
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
		ObjectID:         req.ObjectID,
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

/*
The Scan method handles scanning objects in a table with pagination support. It expects a JSON payload with tenant ID, table hash, optional limit, and next token.
It retrieves the objects from DynamoDB using the ScanTable method, which returns a list of objects and a next token for pagination.
For each object, it generates a presigned GET URL if the object is stored in S3.
Finally, it responds with a list of objects and the next token for pagination.
*/
func (h *ObjectHandler) Scan(c *gin.Context) {
	ctx := c.Request.Context()
	tenantId := c.GetString("tenantId")

	var req objects.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nextToken := ""
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	result, err := h.Dynamo.ScanTable(ctx, tenantId, req.TableHash, req.Limit, nextToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan objects from DynamoDB: " + err.Error()})
		return
	}

	var resp objects.ScanResponse
	for _, obj := range result.Objects {
		resp.Objects = append(resp.Objects, objects.ResultObject{
			ObjectID:         obj.SK[len("OBJ#"):],
			GetURL:           obj.S3Key,
			EncryptedBlob:    obj.EncryptedBlob,
			KMSWrappedDEK:    obj.KMSWrappedDEK,
			MasterWrappedDEK: obj.MasterWrappedDEK,
			DEKNonce:         obj.DEKNonce,
			CreatedAt:        obj.CreatedAt,
			UpdatedAt:        obj.UpdatedAt,
			Version:          obj.Version,
		})
	}
	resp.NextToken = result.NextToken

	c.JSON(http.StatusOK, resp)
}

// Helper function to generate S3 key
func generateS3Key(tenantId string, tableHash string, objectId string, version int32) string {
	return "tenant-" + tenantId + "/" + tableHash + "/" + objectId + "/v" + fmt.Sprintf("%d", version)
}
