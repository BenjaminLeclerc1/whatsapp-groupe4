package auth

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("whatsapp_secret_key") // In production, use os.Getenv("JWT_SECRET")

func GenerateToken(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token valid for 24h
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}