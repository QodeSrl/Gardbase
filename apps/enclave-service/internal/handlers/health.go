package handlers

import (
	"encoding/json"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

func HandleHealth(encoder *json.Encoder, startTime time.Time) {
	uptime := time.Since(startTime).String()
	res := utils.Response{
		Success: true,
		Data: HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now(),
			Uptime:    uptime,
		},
		Message: "Service is healthy",
	}
	utils.SendResponse(encoder, res)
}