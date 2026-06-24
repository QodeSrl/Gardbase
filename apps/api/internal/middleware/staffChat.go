package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaffChatMiddleware protects the staff-facing support chat routes. Staff
// authenticate with a shared bearer token (STAFF_CHAT_TOKEN).
//
// Browsers cannot set custom headers when opening a WebSocket, so the token is
// accepted either via the Authorization header ("Bearer <token>", used by the
// REST endpoints) or via a "token" query parameter (used by the WebSocket
// upgrade request).
func StaffChatMiddleware(staffToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if staffToken == "" {
			c.AbortWithStatusJSON(503, gin.H{"error": "Staff chat is not configured"})
			return
		}

		provided := c.Query("token")
		if provided == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				provided = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(staffToken)) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"error": "Invalid staff token"})
			return
		}

		c.Set("chatRole", "staff")
		c.Next()
	}
}
