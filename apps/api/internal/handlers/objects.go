package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/QodeSrl/gardbase-api/internal/models"
	"github.com/QodeSrl/gardbase-api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateObjectHandler struct {
	S3Client *storage.S3Client
	Dynamo *storage.DynamoClient
	PresignTTL time.Duration
}

func NewCreateObjectHandler(s3Client *storage.S3Client, dynamo *storage.DynamoClient) *CreateObjectHandler {
	return &CreateObjectHandler{
		S3Client: s3Client,
		Dynamo: dynamo,
		PresignTTL: 15 * time.Minute,
	}
}

func (h *CreateObjectHandler) CreateObject(c *gin.Context) {
	ctx := c.Request.Context()
	var req models.CreateObjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{ "error": err.Error() })
		return
	}

	objectId := uuid.NewString()
	s3Key := generateS3Key(req.TenantID, objectId, 1)

	obj := models.NewObject(req.TenantID, objectId, s3Key, req.EncryptedDEK)
	if req.Sensitivity != "" {
		obj.Sensitivity = req.Sensitivity
	} else {
		obj.Sensitivity = models.SensitivityLow
	}
	obj.Status = models.StatusPending
	obj.TTL = time.Now().Add(h.PresignTTL).Unix()

	uploadUrl, err := h.S3Client.PresignPutObjectUrl(ctx, s3Key, h.PresignTTL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ "error": "Failed to generate presigned URL: " + err.Error() })
		return
	}

	if err := h.Dynamo.CreateObjectWithIndexes(ctx, obj, req.Indexes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{ "error": "Failed to create object in DynamoDB: " + err.Error() })
		return
	}

	resp := models.CreateObjectResponse{
		ObjectID: objectId,
		S3Key: s3Key,
		UploadURL: uploadUrl,
		ExpiresIn: int64(h.PresignTTL.Seconds()),
	}

	c.JSON(http.StatusCreated, resp)
}

func FinalizeObject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not implemented yet",
	})
}

func generateS3Key(tenantId string, objectId string, version int32) string {
	return "tenant-" + tenantId + "/objects/" + objectId + "/v" + fmt.Sprintf("%d", version)
}

func GetObject(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"message": "Not implemented yet",
	})
}