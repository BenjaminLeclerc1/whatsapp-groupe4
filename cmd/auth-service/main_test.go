package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRefreshTokenFlow(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Reset in-memory stores for deterministic test
	mu.Lock()
	accountsByID = make(map[string]account)
	accountsByEmail = make(map[string]string)
	refreshTokens = make(map[string]*refreshTokenRecord)
	mu.Unlock()

	jwtSecret := "test-secret"
	accessTTL := 1 * time.Hour
	refreshTTL := 24 * time.Hour

	r := gin.New()
	api := r.Group("/api/v1/auth")
	{
		api.POST("/register", register(jwtSecret, accessTTL, refreshTTL))
		api.POST("/refresh", refresh(jwtSecret, accessTTL, refreshTTL))
		api.POST("/logout", logout())
	}

	// Register
	registerBody := map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "password123",
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", mustJSON(t, registerBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var registerResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &registerResp); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	refresh1, _ := registerResp["refresh_token"].(string)
	if refresh1 == "" {
		t.Fatalf("expected refresh_token in register response")
	}
	token1, _ := registerResp["token"].(string)
	if token1 == "" {
		t.Fatalf("expected token in register response")
	}

	// Refresh: should rotate
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", mustJSON(t, map[string]any{"refresh_token": refresh1}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var refreshResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &refreshResp); err != nil {
		t.Fatalf("unmarshal refresh response: %v", err)
	}
	refresh2, _ := refreshResp["refresh_token"].(string)
	if refresh2 == "" || refresh2 == refresh1 {
		t.Fatalf("expected rotated refresh_token")
	}
	token2, _ := refreshResp["token"].(string)
	if token2 == "" || token2 == token1 {
		// token MAY differ even if claims are same; it should typically change due to iat.
		t.Fatalf("expected new token")
	}

	// Refresh again with old refresh should fail (revoked)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", mustJSON(t, map[string]any{"refresh_token": refresh1}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}

	// Logout revokes current refresh
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", mustJSON(t, map[string]any{"refresh_token": refresh2}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Refresh with revoked refresh should fail
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", mustJSON(t, map[string]any{"refresh_token": refresh2}))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func mustJSON(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return bytes.NewReader(b)
}
