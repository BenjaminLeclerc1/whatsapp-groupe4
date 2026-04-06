package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeMsgPinger struct {
	err error
}

func (f *fakeMsgPinger) Ping(ctx context.Context) error {
	return f.err
}

func TestMessageHealthHandler_Connected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", messageHealthHandler(&fakeMsgPinger{err: nil}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["database"] != "connected" {
		t.Errorf("got %v", body["database"])
	}
}

func TestMessageHealthHandler_Disconnected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", messageHealthHandler(&fakeMsgPinger{err: errors.New("down")}))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["database"] != "disconnected" {
		t.Errorf("got %v", body["database"])
	}
}

// ─── getEnv ───────────────────────────────────────────────────────────────────

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("MSG_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("MSG_TEST_VAR", "set_value")
	defer os.Unsetenv("MSG_TEST_VAR")
	result := getEnv("MSG_TEST_VAR", "default")
	if result != "set_value" {
		t.Errorf("expected 'set_value', got '%s'", result)
	}
}

func TestGetEnv_EmptyFallback(t *testing.T) {
	os.Setenv("MSG_EMPTY_VAR", "")
	defer os.Unsetenv("MSG_EMPTY_VAR")
	result := getEnv("MSG_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestGetEnv_Port(t *testing.T) {
	result := getEnv("PORT", "8082")
	if result != "8082" {
		t.Errorf("expected '8082' as default port, got '%s'", result)
	}
}

func TestGetEnv_PortOverride(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")
	if got := getEnv("PORT", "8082"); got != "9090" {
		t.Errorf("expected override 9090, got %q", got)
	}
}

// ─── requireEnv — chemin succès ───────────────────────────────────────────────

func TestRequireEnv_Set(t *testing.T) {
	os.Setenv("MSG_REQ_VAR", "hello")
	defer os.Unsetenv("MSG_REQ_VAR")
	result := requireEnv("MSG_REQ_VAR")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

// ─── initDB — validation de config ───────────────────────────────────────────

func TestInitDB_InvalidURL(t *testing.T) {
	_, err := initDB("not-a-valid-url")
	if err == nil {
		t.Error("expected error for invalid database URL")
	}
}

func TestInitDB_ParseConfigCoverage(t *testing.T) {
	// Couvre les lignes de config (MaxConns, etc.) puis échoue sur Ping
	_, err := initDB("postgres://user:pass@127.0.0.1:1/db?connect_timeout=1&sslmode=disable")
	if err == nil {
		t.Error("expected error for unreachable host")
	}
}
