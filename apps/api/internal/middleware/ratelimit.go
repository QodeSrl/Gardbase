package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RateLimitMiddleware(logger *zap.Logger) gin.HandlerFunc {
	var mu sync.Mutex
	clients := make(map[string][]time.Time)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		mu.Lock()

		requests := clients[clientIP]
		valid := requests[:0]
		for _, t := range requests {
			if now.Sub(t) < time.Minute {
				valid = append(valid, t)
			}
		}
		clients[clientIP] = append(valid, now)
		count := len(clients[clientIP])

		mu.Unlock()

		if count > 1000 {
			logger.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.Int("requests", count),
			)
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}

		c.Next()
	}
}
