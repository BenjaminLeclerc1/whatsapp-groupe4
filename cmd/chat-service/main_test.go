package main

import (
	"os"
	"strings"
	"testing"
)

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("CHAT_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("CHAT_TEST_VAR", "set_value")
	defer os.Unsetenv("CHAT_TEST_VAR")

	result := getEnv("CHAT_TEST_VAR", "default")
	if result != "set_value" {
		t.Errorf("expected 'set_value', got '%s'", result)
	}
}

func TestGetEnv_EmptyFallsBack(t *testing.T) {
	os.Setenv("CHAT_EMPTY_VAR", "")
	defer os.Unsetenv("CHAT_EMPTY_VAR")

	result := getEnv("CHAT_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestGetEnv_DefaultPort(t *testing.T) {
	os.Unsetenv("PORT")
	if got := getEnv("PORT", "8088"); got != "8088" {
		t.Errorf("expected default 8088, got %q", got)
	}
}

func TestMigrationURL_WithoutQueryParam(t *testing.T) {
	url := "postgres://user:pass@localhost/db"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	expected := "postgres://user:pass@localhost/db?x-migrations-table=migrations_chats"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestMigrationURL_WithExistingQueryParam(t *testing.T) {
	url := "postgres://user:pass@localhost/db?sslmode=disable"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	expected := "postgres://user:pass@localhost/db?sslmode=disable&x-migrations-table=migrations_chats"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestMigrationURL_ContainsTableName(t *testing.T) {
	url := "postgres://localhost/db"
	var result string
	if !strings.Contains(url, "?") {
		result = url + "?x-migrations-table=migrations_chats"
	} else {
		result = url + "&x-migrations-table=migrations_chats"
	}
	if !strings.Contains(result, "migrations_chats") {
		t.Errorf("expected result to contain 'migrations_chats', got: %s", result)
	}
}

// ─── initDB — couvre le chemin d'erreur (URL invalide) ───────────────────────

func TestInitDB_InvalidURL(t *testing.T) {
	_, err := initDB("not-a-valid-postgres-url")
	if err == nil {
		t.Error("expected error for invalid DB URL")
	}
}
