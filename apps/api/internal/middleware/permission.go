package middleware

import "github.com/gin-gonic/gin"

func PermissionMiddleware(requiredPermissions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// extract permissions from context (set by tenant middleware)
		permissions, exists := c.Get("permissions")
		if !exists {
			c.AbortWithStatusJSON(500, gin.H{"error": "Permissions not found in context"})
			return
		}
		userPermissions, ok := permissions.([]string)
		if !ok {
			c.AbortWithStatusJSON(500, gin.H{"error": "Invalid permissions format"})
			return
		}
		// check if user has required permissions
		permSet := make(map[string]struct{}, len(userPermissions))
		for _, p := range userPermissions {
			permSet[p] = struct{}{}
		}
		for _, required := range requiredPermissions {
			if _, ok := permSet[required]; !ok {
				c.AbortWithStatusJSON(403, gin.H{"error": "Forbidden"})
				return
			}
		}
		c.Next()
	}
}
