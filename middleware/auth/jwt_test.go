package auth

import (
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken_Success(t *testing.T) {
	token, err := GenerateToken("user-123", "user", "my-secret-key")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	// Un JWT a 3 parties séparées par des points
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestGenerateToken_EmptySecret(t *testing.T) {
	_, err := GenerateToken("user-123", "user", "")
	if err == nil {
		t.Error("expected error for empty secret, got nil")
	}
}

func TestGenerateToken_AdminRole(t *testing.T) {
	token, err := GenerateToken("admin-1", "admin", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Parse et vérifie les claims
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte("secret"), nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}
	if claims["role"] != "admin" {
		t.Errorf("expected role 'admin', got '%v'", claims["role"])
	}
	if claims["user_id"] != "admin-1" {
		t.Errorf("expected user_id 'admin-1', got '%v'", claims["user_id"])
	}
}

func TestGenerateToken_ClaimsUserID(t *testing.T) {
	token, err := GenerateToken("user-456", "user", "another-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte("another-secret"), nil
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	claims := parsed.Claims.(jwt.MapClaims)
	if claims["user_id"] != "user-456" {
		t.Errorf("unexpected user_id: %v", claims["user_id"])
	}
}

func TestGenerateToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken("user-1", "user", "correct-secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tenter de parser avec un mauvais secret doit échouer
	_, err = jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong-secret"), nil
	})
	if err == nil {
		t.Error("expected error when parsing with wrong secret")
	}
}

func TestGenerateToken_DifferentUsers(t *testing.T) {
	token1, _ := GenerateToken("user-1", "user", "secret")
	token2, _ := GenerateToken("user-2", "user", "secret")

	if token1 == token2 {
		t.Error("tokens for different users should be different")
	}
}
