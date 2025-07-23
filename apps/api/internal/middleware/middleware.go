package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*") // todo: restrict this in prod
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Expose-Headers", "Content-Length")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // No Content
			return
		}

		c.Next()
	}
}

func GinZapLogger(logger *zap.Logger) gin.HandlerFunc {
	return func (c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next() // process request

		latency := time.Since(start)
		
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.String("latency", latency.String()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if raw != "" {
			fields = append(fields, zap.String("query", raw))
		}

		status := c.Writer.Status()
		switch {
			case status >= 400 && status < 500:
				fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
				logger.Warn("client error", fields...)
			case status >= 500:
				fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
				logger.Error("server error", fields...)
			default:
				logger.Info("request completed", fields...)
		}
	}
}

func GinZapRecovery(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				
				fields := []zap.Field{
					zap.Any("error", err),
					zap.ByteString("request", httpRequest),
				}
				
				if stack {
					fields = append(fields, zap.ByteString("stack", debug.Stack()))
				}

				if brokenPipe {
					logger.Error("Broken pipe", fields...)
					c.Error(err.(error))
					c.Abort()
					return
				}

				logger.Error("Panic recovered", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

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