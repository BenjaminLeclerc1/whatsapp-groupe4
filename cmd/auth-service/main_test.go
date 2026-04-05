package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ─── getEnv ───────────────────────────────────────────────────────────────────

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("AUTH_TEST_NONEXISTENT", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("AUTH_TEST_VAR", "hello")
	defer os.Unsetenv("AUTH_TEST_VAR")
	result := getEnv("AUTH_TEST_VAR", "fallback")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestGetEnv_Empty(t *testing.T) {
	os.Setenv("AUTH_EMPTY_VAR", "")
	defer os.Unsetenv("AUTH_EMPTY_VAR")
	result := getEnv("AUTH_EMPTY_VAR", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

// ─── getEnvDuration ───────────────────────────────────────────────────────────

func TestGetEnvDuration_Default(t *testing.T) {
	d := getEnvDuration("AUTH_DURATION_NONEXISTENT", 5*time.Minute)
	if d != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d)
	}
}

func TestGetEnvDuration_ValidValue(t *testing.T) {
	os.Setenv("AUTH_TTL", "1h")
	defer os.Unsetenv("AUTH_TTL")
	d := getEnvDuration("AUTH_TTL", 5*time.Minute)
	if d != time.Hour {
		t.Errorf("expected 1h, got %v", d)
	}
}

func TestGetEnvDuration_InvalidValue(t *testing.T) {
	os.Setenv("AUTH_TTL_INVALID", "notaduration")
	defer os.Unsetenv("AUTH_TTL_INVALID")
	d := getEnvDuration("AUTH_TTL_INVALID", 10*time.Minute)
	if d != 10*time.Minute {
		t.Errorf("expected default 10m, got %v", d)
	}
}

func TestGetEnvDuration_ZeroValue(t *testing.T) {
	os.Setenv("AUTH_TTL_ZERO", "0s")
	defer os.Unsetenv("AUTH_TTL_ZERO")
	d := getEnvDuration("AUTH_TTL_ZERO", 10*time.Minute)
	if d != 10*time.Minute {
		t.Errorf("expected default for zero duration, got %v", d)
	}
}

// ─── normalizeEmail ───────────────────────────────────────────────────────────

func TestNormalizeEmail_Lowercase(t *testing.T) {
	result := normalizeEmail("TEST@EXAMPLE.COM")
	if result != "test@example.com" {
		t.Errorf("expected 'test@example.com', got '%s'", result)
	}
}

func TestNormalizeEmail_TrimSpaces(t *testing.T) {
	result := normalizeEmail("  user@domain.fr  ")
	if result != "user@domain.fr" {
		t.Errorf("expected 'user@domain.fr', got '%s'", result)
	}
}

func TestNormalizeEmail_Mixed(t *testing.T) {
	result := normalizeEmail("  User@Domain.COM  ")
	if result != "user@domain.com" {
		t.Errorf("expected 'user@domain.com', got '%s'", result)
	}
}

// ─── hashRefreshToken ─────────────────────────────────────────────────────────

func TestHashRefreshToken_Deterministic(t *testing.T) {
	plain := "my-refresh-token"
	h1 := hashRefreshToken(plain)
	h2 := hashRefreshToken(plain)
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}
}

func TestHashRefreshToken_DifferentInputs(t *testing.T) {
	h1 := hashRefreshToken("token-a")
	h2 := hashRefreshToken("token-b")
	if h1 == h2 {
		t.Error("expected different hashes for different inputs")
	}
}

func TestHashRefreshToken_HexOutput(t *testing.T) {
	h := hashRefreshToken("any-token")
	if len(h) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Errorf("expected 64 hex chars, got %d", len(h))
	}
}

// ─── generateRefreshToken ─────────────────────────────────────────────────────

func TestGenerateRefreshToken_Success(t *testing.T) {
	plain, hash, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain == "" {
		t.Error("expected non-empty plain token")
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestGenerateRefreshToken_PlainIsHashed(t *testing.T) {
	plain, hash, _ := generateRefreshToken()
	computed := hashRefreshToken(plain)
	if computed != hash {
		t.Error("hash should match hashRefreshToken(plain)")
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	plain1, _, _ := generateRefreshToken()
	plain2, _, _ := generateRefreshToken()
	if plain1 == plain2 {
		t.Error("expected unique refresh tokens")
	}
}

// ─── generateJWT ─────────────────────────────────────────────────────────────

func TestGenerateJWT_Success(t *testing.T) {
	tokenStr, err := generateJWT("user-1", "user@test.com", "secret", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestGenerateJWT_ParseClaims(t *testing.T) {
	tokenStr, err := generateJWT("user-42", "alice@test.com", "my-secret", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("my-secret"), nil
	})
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	claims := parsed.Claims.(*Claims)
	if claims.UserID != "user-42" {
		t.Errorf("expected user-42, got %s", claims.UserID)
	}
	if claims.Email != "alice@test.com" {
		t.Errorf("expected alice@test.com, got %s", claims.Email)
	}
}

func TestGenerateJWT_WrongSecret(t *testing.T) {
	tokenStr, _ := generateJWT("u1", "u@test.com", "correct", time.Hour)
	_, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("wrong"), nil
	})
	if err == nil {
		t.Error("expected error with wrong secret")
	}
}

// ─── authMiddleware ───────────────────────────────────────────────────────────

func setupAuthRouter(secret string) *gin.Engine {
	r := gin.New()
	r.GET("/protected", authMiddleware(secret), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	r := setupAuthRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ShortHeader(t *testing.T) {
	r := setupAuthRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "short")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongPrefix(t *testing.T) {
	r := setupAuthRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	r := setupAuthRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "test-secret"
	tokenStr, err := generateJWT("user-99", "test@test.com", secret, time.Hour)
	if err != nil {
		t.Fatalf("failed to generate JWT: %v", err)
	}

	r := setupAuthRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := "test-secret"
	// TTL négatif → token déjà expiré
	tokenStr, _ := generateJWT("user-1", "u@test.com", secret, -time.Hour)

	r := setupAuthRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}
