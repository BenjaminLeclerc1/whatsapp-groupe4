package middleware

import (
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func IsValidUUID(s string) bool {
	return uuidRe.MatchString(s)
}

// ParsePagination extracts and validates cursor-based pagination parameters
// from query string. Returns (cursor, limit) with safe defaults.
func ParsePagination(c *gin.Context) (cursor string, limit int) {
	cursor = c.Query("cursor")
	if cursor != "" && !IsValidUUID(cursor) {
		cursor = ""
	}

	limit = 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	return cursor, limit
}
