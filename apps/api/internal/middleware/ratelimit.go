package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RateLimitMiddleware(logger *zap.Logger) gin.HandlerFunc {
	clients := make(map[string][]time.Time)
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		if requests, exists := clients[clientIP]; exists {
			validRequests := []time.Time{}
			for _, t := range requests {
				if now.Sub(t) < time.Minute {
					validRequests = append(validRequests, t)
				}
			}
			clients[clientIP] = validRequests
		}

		// limit to 100 reqs per min
		if len(clients[clientIP]) >= 100 {
			logger.Warn("Rate limit exceeded", zap.String("client_ip", clientIP), zap.Int("requests", len(clients[clientIP])))
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		clients[clientIP] = append(clients[clientIP], now)

		c.Next()
	}
}