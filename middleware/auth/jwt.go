package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken signs a JWT with the given secret (from env in the calling service, never hardcoded).
func GenerateToken(userID, role, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("JWT secret is required")
	}
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
