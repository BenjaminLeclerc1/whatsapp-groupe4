package main

import (
	"os"
	"testing"
)

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
