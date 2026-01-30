package healthCheck

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/services"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/gin-gonic/gin"
)

type HealthCheckHandler struct {
	Vsock    *services.Vsock
	S3Client *storage.S3Client
	Dynamo   *storage.DynamoClient
	KMS      *services.KMS
}

func (h *HealthCheckHandler) HandleAPIHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"message": "The service is running smoothly.",
	})
}

func (h *HealthCheckHandler) HandleEnclaveHealthCheck(c *gin.Context) {
	req := enclaveproto.Request{
		Type:    "health",
		Payload: nil,
	}
	resBytes, err := h.Vsock.SendToEnclave(req, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var res enclaveproto.Response[json.RawMessage]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal health response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	c.JSON(200, gin.H{
		"status":  "healthy",
		"message": "Enclave is healthy",
		"data":    res.Data,
	})
}

func (h *HealthCheckHandler) HandleStorageHealthCheck(c *gin.Context) {
	s3Healthy := false
	dynamoHealthy := false

	// Check S3 connectivity
	err := h.S3Client.TestConnnectivity(c.Request.Context())
	if err == nil {
		s3Healthy = true
	}
	// Check DynamoDB connectivity
	err = h.Dynamo.TestConnnectivity(c.Request.Context())
	if err == nil {
		dynamoHealthy = true
	}

	var status string
	if s3Healthy && dynamoHealthy {
		status = "healthy"
	} else {
		status = "unhealthy"
	}

	c.JSON(200, gin.H{
		"status":         status,
		"s3_healthy":     s3Healthy,
		"dynamo_healthy": dynamoHealthy,
	})
}

func (h *HealthCheckHandler) HandleKMSHealthCheck(c *gin.Context) {
	err := h.KMS.TestConnectivity(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}
