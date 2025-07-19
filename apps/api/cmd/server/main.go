package main

import (
	"github.com/QodeSrl/gardbase-api/internal/handlers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	r := gin.Default()

	r.GET("/health", handlers.HealthCheckHandler)

	r.Run(":8080")
}