package middleware

import (
	"context"
	"regexp"

	"github.com/gin-gonic/gin"
)

type tenantKeyType string

func TenantMiddleware() gin.HandlerFunc {
	tenantIdRegex := regexp.MustCompile(`^[a-z0-9-]{3,64}$`)
	return func(c *gin.Context) {
		tenantId := c.GetHeader("X-Tenant-ID")
		// TOOD: check registered tenants
		if !tenantIdRegex.MatchString(tenantId) {
			c.AbortWithStatusJSON(400, gin.H{"error": "Invalid X-Tenant-ID header"})
		}
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
