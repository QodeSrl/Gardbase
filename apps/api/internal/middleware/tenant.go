package middleware

import (
	"context"
	"regexp"

	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/gin-gonic/gin"
)

type tenantKeyType string
type permissionsKeyType string

func TenantMiddleware(dynamoClient *storage.DynamoClient) gin.HandlerFunc {
	tenantIdRegex := regexp.MustCompile(`^[a-z0-9-]{3,64}$`)
	return func(c *gin.Context) {
		tenantId := c.GetHeader("X-Tenant-ID")
		apiKey := c.GetHeader("X-API-Key")
		if tenantId == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "X-Tenant-ID header is required"})
			return
		}
		if !tenantIdRegex.MatchString(tenantId) {
			c.AbortWithStatusJSON(400, gin.H{"error": "Invalid X-Tenant-ID header"})
		}
		if apiKey == "" {
			c.AbortWithStatusJSON(400, gin.H{"error": "X-API-Key header is required"})
			return
		}
		apiKeyRecord, err := dynamoClient.FindAPIKey(c.Request.Context(), tenantId, apiKey)
		if err != nil || apiKeyRecord == nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid API key"})
			return
		}

		c.Set("tenantId", tenantId)
		c.Set("permissions", apiKeyRecord.Permissions)

		// TODO: use permissions to restrict access to certain endpoints
		ctx := context.WithValue(c.Request.Context(), tenantKeyType("tenantId"), tenantId)
		ctx = context.WithValue(ctx, permissionsKeyType("permissions"), apiKeyRecord.Permissions)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
