package handlers

import (
	"time"

	"github.com/qodesrl/gardbase/pkg/enclaveproto"
)

func HandleHealth(startTime time.Time) enclaveproto.HealthResponse {
	uptime := time.Since(startTime).String()
	res := enclaveproto.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    uptime,
	}
	return res
}
