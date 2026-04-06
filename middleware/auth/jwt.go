// package auth

// import (
// 	"time"
// 	"github.com/golang-jwt/jwt/v5"
// )

// var secretKey = []byte("whatsapp_secret_key") // In production, use os.Getenv("JWT_SECRET")

// func GenerateToken(userID, role string) (string, error) {
// 	claims := jwt.MapClaims{
// 		"user_id": userID,
// 		"role":    role,
// 		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token valid for 24h
// 	}

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
// 	return token.SignedString(secretKey)
// }


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
    "os"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID, role string) (string, error) {
    // Get the secret from the environment (Docker)
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        // Fallback for local testing, but make sure it matches docker-compose!
        secret = "whatsapp-groupe4-secret-default" 
    }
    
    var secretKey = []byte(secret)

    claims := jwt.MapClaims{
        "user_id": userID,
        "role":    role,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(secretKey)
}
