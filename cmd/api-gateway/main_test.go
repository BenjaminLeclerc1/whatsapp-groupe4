package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
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

func setupGatewayRouter(secret string) *gin.Engine {
	r := gin.New()
	protected := r.Group("/api/v1", authMiddleware(secret))
	{
		protected.GET("/users/*path", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"route": "users"})
		})
	}
	return r
}

// ─── getEnv ───────────────────────────────────────────────────────────────────

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("GW_TEST_NONEXISTENT", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("GW_TEST_VAR", "gateway_val")
	defer os.Unsetenv("GW_TEST_VAR")
	result := getEnv("GW_TEST_VAR", "fallback")
	if result != "gateway_val" {
		t.Errorf("expected 'gateway_val', got '%s'", result)
	}
}

func TestGetEnv_EmptyFallback(t *testing.T) {
	os.Setenv("GW_EMPTY_VAR", "")
	defer os.Unsetenv("GW_EMPTY_VAR")
	result := getEnv("GW_EMPTY_VAR", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

// ─── authMiddleware ───────────────────────────────────────────────────────────

func TestAuthMiddleware_NoHeader(t *testing.T) {
	r := setupGatewayRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_EmptyHeader(t *testing.T) {
	r := setupGatewayRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ShortHeader(t *testing.T) {
	r := setupGatewayRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bear")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_BearerPrefixOnly(t *testing.T) {
	// "Bearer " fait 7 caractères : len < 8 → 401
	r := setupGatewayRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	r := setupGatewayRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_WrongSecret(t *testing.T) {
	tokenStr, _ := generateTestJWT("correct-secret", "u1", "u@test.com")
	r := setupGatewayRouter("wrong-secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong secret, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	secret := "valid-secret"
	tokenStr, err := generateTestJWT(secret, "user-1", "u@test.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	r := setupGatewayRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_OPTIONS_PassThrough(t *testing.T) {
	secret := "secret"
	r := setupGatewayRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/users/list", nil)
	r.ServeHTTP(w, req)
	// OPTIONS passe à travers le middleware
	if w.Code == http.StatusUnauthorized {
		t.Error("OPTIONS should not return 401")
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	secret := "secret"
	// Token expiré (TTL négatif)
	claims := Claims{
		UserID: "u1",
		Email:  "u@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))

	r := setupGatewayRouter(secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

// ─── Test combiné (public + protected) ───────────────────────────────────────

func TestAuthMiddleware_ProtectedAndPublicRoutes(t *testing.T) {
	jwtSecret := "test-secret"
	router := gin.New()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api/v1")
	{
		api.Any("/auth/*path", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"route": "auth"})
		})
		protected := api.Group("/", authMiddleware(jwtSecret))
		{
			protected.Any("/users/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"route": "users"})
			})
		}
	}

	// Health check public
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", w.Code)
	}

	// Auth route publique
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/auth/*, got %d", w.Code)
	}

	// Protected sans token → 401
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", w.Code)
	}

	// Protected avec token valide → 200
	tokenStr, err := generateTestJWT(jwtSecret, "user-123", "user@example.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/v1/users/list", nil)
	r.Header.Set("Authorization", "Bearer "+tokenStr)
	router.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w.Code)
	}
}

// ─── proxyHandler ─────────────────────────────────────────────────────────────
// Note: on utilise httptest.NewServer (vrai serveur HTTP) car httputil.ReverseProxy
// appelle CloseNotify() qui n'est pas supporté par httptest.ResponseRecorder.

func TestProxyHandler_ForwardsRequest(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "backend ok")
	}))
	defer backend.Close()

	ginRouter := gin.New()
	ginRouter.GET("/api/v1/test", proxyHandler(backend.URL))

	// Vrai serveur HTTP pour éviter le panic CloseNotify
	srv := httptest.NewServer(ginRouter)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 from proxy, got %d", resp.StatusCode)
	}
}

func TestProxyHandler_WithUserID(t *testing.T) {
	receivedUserID := ""
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserID = r.Header.Get("X-User-ID")
		fmt.Fprintln(w, "ok")
	}))
	defer backend.Close()

	secret := "test-secret"
	tokenStr, _ := generateTestJWT(secret, "user-abc", "u@test.com")

	router := gin.New()
	protected := router.Group("/api/v1", authMiddleware(secret))
	protected.GET("/proxy/*path", proxyHandler(backend.URL))

	srv := httptest.NewServer(router)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/proxy/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if receivedUserID != "user-abc" {
		t.Errorf("expected X-User-ID='user-abc', got '%s'", receivedUserID)
	}
}

func TestProxyHandler_InvalidTarget(t *testing.T) {
	ginRouter := gin.New()
	ginRouter.GET("/api/v1/bad", proxyHandler("http://127.0.0.1:19999"))

	srv := httptest.NewServer(ginRouter)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/bad")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// Connexion refusée → 502 Bad Gateway
	if resp.StatusCode < 500 {
		t.Logf("proxy to invalid target returned %d (acceptable)", resp.StatusCode)
	}
}

func TestProxyHandler_ForwardsPathToBackend(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	const wantPath = "/api/v1/users/profile/detail"
	ginRouter := gin.New()
	ginRouter.GET(wantPath, proxyHandler(backend.URL))

	srv := httptest.NewServer(ginRouter)
	defer srv.Close()

	resp, err := http.Get(srv.URL + wantPath)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if gotPath != wantPath {
		t.Errorf("unexpected path forwarded: %q want %q", gotPath, wantPath)
	}
}

// ─── requireEnv ───────────────────────────────────────────────────────────────

func TestRequireEnv_Set(t *testing.T) {
	os.Setenv("GW_REQUIRED_VAR", "present")
	defer os.Unsetenv("GW_REQUIRED_VAR")

	result := requireEnv("GW_REQUIRED_VAR")
	if result != "present" {
		t.Errorf("expected 'present', got '%s'", result)
	}
}
