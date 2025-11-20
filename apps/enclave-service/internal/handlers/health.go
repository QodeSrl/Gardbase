package handlers

import (
	"encoding/json"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
)

func HandleHealth(encoder *json.Encoder, startTime time.Time) {
	uptime := time.Since(startTime).String()
	res := enclaveproto.Response{
		Success: true,
		Data: enclaveproto.HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now(),
			Uptime:    uptime,
		},
		Message: "Service is healthy",
	}
	utils.SendResponse(encoder, res)
}
