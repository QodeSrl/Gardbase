package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

type tenantKeyType string

func TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantId := c.GetHeader("X-Tenant-ID")
		if tenantId == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "X-Tenant-ID header is required"})
			return
		}
		c.Set("tenantId", tenantId)

		ctx := context.WithValue(c.Request.Context(), tenantKeyType("tenantId"), tenantId)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
