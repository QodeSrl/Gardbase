package middleware

import "github.com/gin-gonic/gin"

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
