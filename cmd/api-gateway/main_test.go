package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestAuthMiddleware_ProtectedAndPublicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	jwtSecret := "test-secret"

	// Health (public)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		// Public auth routes
		api.Any("/auth/*path", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"route": "auth"})
		})

		// Protected routes
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"route": "users"})
			})
		}
	}

	// 1) Public /health sans token
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", w.Code)
	}

	// 2) Public /api/v1/auth sans token
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/auth/*, got %d", w.Code)
	}

	// 3) Protected /api/v1/users sans token -> 401
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for protected route without token, got %d", w.Code)
	}

	// 4) Protected avec token valide -> 200
	tokenStr, err := generateTestJWT(jwtSecret, "user-123", "user@example.com")
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for protected route with valid token, got %d", w.Code)
	}
}

func generateTestJWT(secret, userID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}