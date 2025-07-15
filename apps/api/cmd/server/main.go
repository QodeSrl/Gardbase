package main

import (
	"github.com/QodeSrl/gardbase-api/internal/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/health", handlers.HealthCheckHandler)

	r.Run(":8080")
}