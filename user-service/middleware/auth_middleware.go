package middleware

import (
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		// 1. Verify token signature
		// 2. Parse claims
		// 3. If invalid, c.AbortWithStatus(401)
		c.Next()
	}
}
