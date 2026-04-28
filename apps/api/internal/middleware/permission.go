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
		for _, required := range requiredPermissions {
			hasPermission := false
			for _, userPerm := range userPermissions {
				if userPerm == required {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				c.AbortWithStatusJSON(403, gin.H{"error": "Forbidden: insufficient permissions"})
				return
			}
		}
		c.Next()
	}
}
