package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ObjectHandler struct {
	S3Client   *storage.S3Client
	Dynamo     *storage.DynamoClient
	PresignTTL time.Duration
}

func NewObjectHandler(s3Client *storage.S3Client, dynamo *storage.DynamoClient) *ObjectHandler {
	return &ObjectHandler{
		S3Client:   s3Client,
		Dynamo:     dynamo,
		PresignTTL: 15 * time.Minute,
	}
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
	var req models.CreateObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tenantId := c.GetString("tenantId")

	objectId := uuid.NewString()
	s3Key := generateS3Key(tenantId, objectId, 1)

	obj := models.NewObject(tenantId, objectId, s3Key, req.EncryptedDEK)
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

	if err := h.Dynamo.CreateObjectWithIndexes(ctx, obj, req.Indexes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create object in DynamoDB: " + err.Error()})
		return
	}

	resp := models.CreateObjectResponse{
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

	obj, err := h.Dynamo.GetObject(ctx, tenantId, id)
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

	resp := models.GetObjectResponse{
		ObjectID:     id,
		EncryptedDEK: obj.EncryptedDEK,
		GetURL:       getUrl,
		CreatedAt:    obj.CreatedAt,
		UpdatedAt:    obj.UpdatedAt,
		Version:      obj.Version,
	}

	c.JSON(http.StatusOK, resp)
}

// Helper function to generate S3 key
func generateS3Key(tenantId string, objectId string, version int32) string {
	return "tenant-" + tenantId + "/objects/" + objectId + "/v" + fmt.Sprintf("%d", version)
}
