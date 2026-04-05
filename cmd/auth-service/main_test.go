package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Mocks
// ═══════════════════════════════════════════════════════════════════════════════

// mockRow implémente rowScanner pour les tests
type mockRow struct {
	vals []any
	err  error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.vals) {
			break
		}
		switch v := d.(type) {
		case *string:
			if s, ok := r.vals[i].(string); ok {
				*v = s
			}
		case *time.Time:
			if t, ok := r.vals[i].(time.Time); ok {
				*v = t
			}
		case **time.Time:
			if r.vals[i] == nil {
				*v = nil
			} else if t, ok := r.vals[i].(time.Time); ok {
				*v = &t
			}
		}
	}
	return nil
}

// mockTx implémente txDB pour les tests
type mockTx struct {
	queryRows   []*mockRow
	execErrors  []error
	commitErr   error
	qIdx, eIdx  int
}

func (m *mockTx) QueryRow(_ context.Context, _ string, _ ...any) rowScanner {
	if m.qIdx < len(m.queryRows) {
		r := m.queryRows[m.qIdx]
		m.qIdx++
		return r
	}
	return &mockRow{err: errors.New("mockTx: no more rows")}
}

func (m *mockTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	if m.eIdx < len(m.execErrors) {
		e := m.execErrors[m.eIdx]
		m.eIdx++
		return pgconn.CommandTag{}, e
	}
	m.eIdx++
	return pgconn.CommandTag{}, nil
}

func (m *mockTx) Commit(_ context.Context) error   { return m.commitErr }
func (m *mockTx) Rollback(_ context.Context) error { return nil }

// mockPool implémente dbPool pour les tests
type mockPool struct {
	execErrors  []error
	queryRows   []*mockRow
	beginTx     txDB
	beginErr    error
	eIdx, qIdx  int
}

func (m *mockPool) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	if m.eIdx < len(m.execErrors) {
		e := m.execErrors[m.eIdx]
		m.eIdx++
		return pgconn.CommandTag{}, e
	}
	m.eIdx++
	return pgconn.CommandTag{}, nil
}

func (m *mockPool) QueryRow(_ context.Context, _ string, _ ...any) rowScanner {
	if m.qIdx < len(m.queryRows) {
		r := m.queryRows[m.qIdx]
		m.qIdx++
		return r
	}
	return &mockRow{err: errors.New("mockPool: no more rows")}
}

func (m *mockPool) Begin(_ context.Context) (txDB, error) {
	return m.beginTx, m.beginErr
}

func (m *mockPool) Ping(_ context.Context) error { return nil }

// ═══════════════════════════════════════════════════════════════════════════════
// Utilitaires de setup
// ═══════════════════════════════════════════════════════════════════════════════

func setupRouterWithPool(pool dbPool, secret string) *gin.Engine {
	ttl := time.Hour
	r := gin.New()
	api := r.Group("/api/v1/auth")
	{
		api.POST("/register", register(pool, secret, ttl, ttl))
		api.POST("/login", login(pool, secret, ttl, ttl))
		api.POST("/refresh", refresh(pool, secret, ttl, ttl))
		api.POST("/logout", logout(pool))
		api.GET("/me", authMiddleware(secret), me(pool))
	}
	return r
}

// setupHandlerRouter utilise nil pour tester les validations (avant appel DB)
func setupHandlerRouter() *gin.Engine {
	r := gin.Default() // Recovery attrape les panics nil-pool
	ttl := time.Hour
	api := r.Group("/api/v1/auth")
	{
		api.POST("/register", register(nil, "secret", ttl, ttl))
		api.POST("/login", login(nil, "secret", ttl, ttl))
		api.POST("/refresh", refresh(nil, "secret", ttl, ttl))
		api.POST("/logout", logout(nil))
	}
	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// ═══════════════════════════════════════════════════════════════════════════════
// getEnv
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("AUTH_TEST_NONEXISTENT", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("AUTH_TEST_VAR", "hello")
	defer os.Unsetenv("AUTH_TEST_VAR")
	if result := getEnv("AUTH_TEST_VAR", "fallback"); result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestGetEnv_Empty(t *testing.T) {
	os.Setenv("AUTH_EMPTY_VAR", "")
	defer os.Unsetenv("AUTH_EMPTY_VAR")
	if result := getEnv("AUTH_EMPTY_VAR", "default"); result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// requireEnv
// ═══════════════════════════════════════════════════════════════════════════════

func TestRequireEnv_Set(t *testing.T) {
	os.Setenv("AUTH_REQ_VAR", "myvalue")
	defer os.Unsetenv("AUTH_REQ_VAR")
	if result := requireEnv("AUTH_REQ_VAR"); result != "myvalue" {
		t.Errorf("expected 'myvalue', got '%s'", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// getEnvDuration
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetEnvDuration_Default(t *testing.T) {
	d := getEnvDuration("AUTH_DUR_NONEXISTENT", 5*time.Minute)
	if d != 5*time.Minute {
		t.Errorf("expected 5m, got %v", d)
	}
}

func TestGetEnvDuration_Valid(t *testing.T) {
	os.Setenv("AUTH_DUR_VAR", "2h")
	defer os.Unsetenv("AUTH_DUR_VAR")
	if d := getEnvDuration("AUTH_DUR_VAR", time.Minute); d != 2*time.Hour {
		t.Errorf("expected 2h, got %v", d)
	}
}

func TestGetEnvDuration_Invalid(t *testing.T) {
	os.Setenv("AUTH_DUR_INVALID", "notaduration")
	defer os.Unsetenv("AUTH_DUR_INVALID")
	if d := getEnvDuration("AUTH_DUR_INVALID", time.Minute); d != time.Minute {
		t.Errorf("expected 1m fallback, got %v", d)
	}
}

func TestGetEnvDuration_Zero(t *testing.T) {
	os.Setenv("AUTH_DUR_ZERO", "0s")
	defer os.Unsetenv("AUTH_DUR_ZERO")
	if d := getEnvDuration("AUTH_DUR_ZERO", time.Hour); d != time.Hour {
		t.Errorf("expected fallback for 0 duration, got %v", d)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// normalizeEmail
// ═══════════════════════════════════════════════════════════════════════════════

func TestNormalizeEmail_AlreadyNormal(t *testing.T) {
	if result := normalizeEmail("user@test.com"); result != "user@test.com" {
		t.Errorf("unexpected: %s", result)
	}
}

func TestNormalizeEmail_Uppercase(t *testing.T) {
	if result := normalizeEmail("USER@Test.COM"); result != "user@test.com" {
		t.Errorf("expected lowercase, got '%s'", result)
	}
}

func TestNormalizeEmail_Spaces(t *testing.T) {
	if result := normalizeEmail("  alice@example.com  "); result != "alice@example.com" {
		t.Errorf("expected trimmed, got '%s'", result)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// hashRefreshToken
// ═══════════════════════════════════════════════════════════════════════════════

func TestHashRefreshToken_Deterministic(t *testing.T) {
	h1 := hashRefreshToken("token123")
	h2 := hashRefreshToken("token123")
	if h1 != h2 {
		t.Error("expected same hash for same input")
	}
}

func TestHashRefreshToken_Different(t *testing.T) {
	h1 := hashRefreshToken("token1")
	h2 := hashRefreshToken("token2")
	if h1 == h2 {
		t.Error("expected different hash for different input")
	}
}

func TestHashRefreshToken_Length(t *testing.T) {
	h := hashRefreshToken("test")
	if len(h) != 64 {
		t.Errorf("expected SHA-256 hex (64 chars), got %d chars", len(h))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// generateRefreshToken
// ═══════════════════════════════════════════════════════════════════════════════

func TestGenerateRefreshToken_Success(t *testing.T) {
	plain, hash, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plain == "" || hash == "" {
		t.Error("expected non-empty plain and hash")
	}
	if plain == hash {
		t.Error("plain and hash should differ")
	}
}

func TestGenerateRefreshToken_HashMatchesPlain(t *testing.T) {
	plain, hash, _ := generateRefreshToken()
	expected := hashRefreshToken(plain)
	if hash != expected {
		t.Error("hash does not match hashRefreshToken(plain)")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// generateJWT
// ═══════════════════════════════════════════════════════════════════════════════

func TestGenerateJWT_Success(t *testing.T) {
	tokenStr, err := generateJWT("user-1", "u@test.com", "secret", time.Hour)
	if err != nil || tokenStr == "" {
		t.Errorf("expected valid JWT, got err=%v", err)
	}
}

func TestGenerateJWT_EmptySecret(t *testing.T) {
	tokenStr, err := generateJWT("u", "u@test.com", "", time.Hour)
	if err != nil || tokenStr == "" {
		t.Errorf("empty secret should still produce token, got err=%v", err)
	}
}

func TestGenerateJWT_ContainsClaims(t *testing.T) {
	secret := "my-secret"
	tokenStr, _ := generateJWT("user-99", "test@test.com", secret, time.Hour)
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("token parse failed: %v", err)
	}
	claims := token.Claims.(*Claims)
	if claims.UserID != "user-99" {
		t.Errorf("expected UserID 'user-99', got '%s'", claims.UserID)
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

// ═══════════════════════════════════════════════════════════════════════════════
// isUniqueViolation
// ═══════════════════════════════════════════════════════════════════════════════

func TestIsUniqueViolation_NilError(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("expected false for nil error")
	}
}

func TestIsUniqueViolation_OtherError(t *testing.T) {
	if isUniqueViolation(errors.New("some generic error")) {
		t.Error("expected false for non-pgconn error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// authMiddleware
// ═══════════════════════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════════════════════
// register handler — validation (nil pool, couverture avant DB)
// ═══════════════════════════════════════════════════════════════════════════════

func TestRegister_MissingFields(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/register", map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRegister_PasswordTooShort(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"username": "alice", "email": "alice@test.com", "password": "123",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", w.Code)
	}
}

func TestRegister_ValidInput_NilPool(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"username": "alice", "email": "alice@example.com", "password": "password123",
	})
	if w.Code == http.StatusBadRequest {
		t.Errorf("valid input should pass validation, not 400")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// register handler — avec mockPool (couverture des chemins DB)
// ═══════════════════════════════════════════════════════════════════════════════

func TestRegister_DBError(t *testing.T) {
	pool := &mockPool{execErrors: []error{nil, errors.New("db error")}}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"username": "bob", "email": "bob@test.com", "password": "secret123",
	})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on DB error, got %d", w.Code)
	}
}

func TestRegister_Success(t *testing.T) {
	// Exec 1: deleteExpiredRefreshTokens → nil
	// Exec 2: INSERT auth_users → nil
	// Exec 3: INSERT auth_refresh_tokens → nil
	pool := &mockPool{execErrors: []error{nil, nil, nil}}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"username": "alice", "email": "alice@test.com", "password": "password123",
	})
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 on success, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestRegister_RefreshTokenInsertError(t *testing.T) {
	// Exec 1: deleteExpiredRefreshTokens → nil
	// Exec 2: INSERT auth_users → nil
	// Exec 3: INSERT auth_refresh_tokens → error
	pool := &mockPool{execErrors: []error{nil, nil, errors.New("refresh insert failed")}}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"username": "carol", "email": "carol@test.com", "password": "password123",
	})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on refresh insert error, got %d", w.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// login handler
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogin_MissingFields(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/login", map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogin_ValidInput_NilPool(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "alice@example.com", "password": "password123",
	})
	if w.Code == http.StatusBadRequest {
		t.Error("valid input should pass validation")
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	pool := &mockPool{
		execErrors: []error{nil}, // deleteExpiredRefreshTokens
		queryRows:  []*mockRow{{err: errors.New("no rows")}},
	}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "unknown@test.com", "password": "password123",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown user, got %d", w.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	realHash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	pool := &mockPool{
		execErrors: []error{nil},
		queryRows: []*mockRow{{vals: []any{
			"user-1", "alice", "alice@test.com", string(realHash), "active", time.Now(),
		}}},
	}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "alice@test.com", "password": "wrong-password",
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	realHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	pool := &mockPool{
		execErrors: []error{nil, nil}, // deleteExpired + INSERT refresh
		queryRows: []*mockRow{{vals: []any{
			"user-1", "alice", "alice@test.com", string(realHash), "active", time.Now(),
		}}},
	}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "alice@test.com", "password": "password123",
	})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on successful login, got %d (body: %s)", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil {
		t.Error("expected 'token' in response")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// logout handler
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogout_MissingToken(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/logout", map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogout_ValidToken_NilPool(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/logout", map[string]string{"refresh_token": "some-token"})
	if w.Code == http.StatusBadRequest {
		t.Error("valid token should pass validation")
	}
}

func TestLogout_Success(t *testing.T) {
	pool := &mockPool{execErrors: []error{nil}}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/logout", map[string]string{"refresh_token": "my-refresh-token"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on logout, got %d", w.Code)
	}
}

func TestLogout_DBError_StillOK(t *testing.T) {
	// Le handler logout ignore l'erreur DB (logger.Error) et retourne toujours 200
	pool := &mockPool{execErrors: []error{errors.New("db error")}}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/logout", map[string]string{"refresh_token": "my-refresh-token"})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 even on DB error (error is logged), got %d", w.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// refresh handler
// ═══════════════════════════════════════════════════════════════════════════════

func TestRefresh_MissingToken(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRefresh_ValidToken_NilPool(t *testing.T) {
	r := setupHandlerRouter()
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "some-token"})
	if w.Code == http.StatusBadRequest {
		t.Error("valid token should pass validation")
	}
}

func TestRefresh_BeginError(t *testing.T) {
	pool := &mockPool{
		execErrors: []error{nil}, // deleteExpiredRefreshTokens
		beginErr:   errors.New("begin failed"),
	}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "some-token"})
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on begin error, got %d", w.Code)
	}
}

func TestRefresh_TokenNotFound(t *testing.T) {
	tx := &mockTx{queryRows: []*mockRow{{err: errors.New("token not found")}}}
	pool := &mockPool{execErrors: []error{nil}, beginTx: tx}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "bad-token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid refresh token, got %d", w.Code)
	}
}

func TestRefresh_TokenRevoked(t *testing.T) {
	revokedAt := time.Now().Add(-time.Hour)
	tx := &mockTx{queryRows: []*mockRow{{vals: []any{
		"user-1", time.Now().Add(time.Hour), revokedAt,
	}}}}
	pool := &mockPool{execErrors: []error{nil}, beginTx: tx}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "revoked-token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for revoked token, got %d", w.Code)
	}
}

func TestRefresh_TokenExpired(t *testing.T) {
	tx := &mockTx{queryRows: []*mockRow{{vals: []any{
		"user-1", time.Now().Add(-time.Hour), nil, // expiresAt in the past
	}}}}
	pool := &mockPool{execErrors: []error{nil}, beginTx: tx}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "expired-token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestRefresh_UserNotFound(t *testing.T) {
	tx := &mockTx{queryRows: []*mockRow{
		{vals: []any{"user-1", time.Now().Add(time.Hour), nil}}, // token ok
		{err: errors.New("user not found")},                     // user query fails
	}}
	pool := &mockPool{execErrors: []error{nil}, beginTx: tx}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": "valid-token"})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 when user not found, got %d", w.Code)
	}
}

func TestRefresh_Success(t *testing.T) {
	tx := &mockTx{
		queryRows: []*mockRow{
			{vals: []any{"user-1", time.Now().Add(time.Hour), nil}},                               // token
			{vals: []any{"alice", "alice@test.com", "active", time.Now().Add(-24 * time.Hour)}},   // user
		},
		execErrors: []error{nil, nil}, // UPDATE old token + INSERT new token
	}
	pool := &mockPool{execErrors: []error{nil}, beginTx: tx}
	r := setupRouterWithPool(pool, "secret")
	w := postJSON(r, "/api/v1/auth/refresh", map[string]string{"refresh_token": strings.Repeat("a", 64)})
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on refresh success, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// me handler
// ═══════════════════════════════════════════════════════════════════════════════

func TestMe_Unauthorized(t *testing.T) {
	pool := &mockPool{}
	r := setupRouterWithPool(pool, "secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestMe_UserNotFound(t *testing.T) {
	secret := "test-secret"
	token, _ := generateJWT("user-1", "u@test.com", secret, time.Hour)
	pool := &mockPool{queryRows: []*mockRow{{err: errors.New("not found")}}}
	r := setupRouterWithPool(pool, secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	secret := "test-secret"
	token, _ := generateJWT("user-1", "alice@test.com", secret, time.Hour)
	pool := &mockPool{queryRows: []*mockRow{{vals: []any{
		"alice", "alice@test.com", "active", time.Now().Add(-24 * time.Hour),
	}}}}
	r := setupRouterWithPool(pool, secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}
