package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "user_id"

// ExtractUserID reads the X-User-ID header set by the API Gateway after JWT
// validation. The channel-service itself does not verify JWTs — that
// responsibility belongs to the gateway. This keeps the service focused and
// avoids duplicating the JWT secret across services.
func ExtractUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-ID header"})
			c.Abort()
			return
		}

		if !IsValidUUID(userID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-User-ID format"})
			c.Abort()
			return
		}

		c.Set(UserIDKey, userID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from the Gin context.
func GetUserID(c *gin.Context) string {
	v, _ := c.Get(UserIDKey)
	s, _ := v.(string)
	return s
}
