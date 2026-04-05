package main

import (
	"os"
	"testing"
)

func TestGetEnvDefault(t *testing.T) {
	result := getEnv("CHAT_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnvSet(t *testing.T) {
	os.Setenv("CHAT_TEST_VAR", "chat_value")
	defer os.Unsetenv("CHAT_TEST_VAR")

	result := getEnv("CHAT_TEST_VAR", "default")
	if result != "chat_value" {
		t.Errorf("expected 'chat_value', got '%s'", result)
	}
}

func TestGetEnvEmpty(t *testing.T) {
	os.Setenv("CHAT_EMPTY_VAR", "")
	defer os.Unsetenv("CHAT_EMPTY_VAR")

	result := getEnv("CHAT_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestRunMigrationsURL_WithoutQuery(t *testing.T) {
	url := "postgres://user:pass@localhost/db"
	expected := url + "?x-migrations-table=migrations_chats"

	var result string
	if len(url) > 0 {
		if !containsQuestion(url) {
			result = url + "?x-migrations-table=migrations_chats"
		} else {
			result = url + "&x-migrations-table=migrations_chats"
		}
	}
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestRunMigrationsURL_WithQuery(t *testing.T) {
	url := "postgres://user:pass@localhost/db?sslmode=disable"
	expected := url + "&x-migrations-table=migrations_chats"

	var result string
	if containsQuestion(url) {
		result = url + "&x-migrations-table=migrations_chats"
	} else {
		result = url + "?x-migrations-table=migrations_chats"
	}
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func containsQuestion(s string) bool {
	for _, c := range s {
		if c == '?' {
			return true
		}
	}
	return false
}
