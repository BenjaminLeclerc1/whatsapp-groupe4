package main

import (
	"testing"
	"os"
)

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("MESSAGE_TEST_VAR_NONEXISTENT", "default_val")
	if result != "default_val" {
		t.Errorf("expected 'default_val', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("MESSAGE_TEST_VAR", "hello")
	defer os.Unsetenv("MESSAGE_TEST_VAR")

	result := getEnv("MESSAGE_TEST_VAR", "default")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestGetEnv_Empty(t *testing.T) {
	os.Setenv("MESSAGE_EMPTY_VAR", "")
	defer os.Unsetenv("MESSAGE_EMPTY_VAR")

	result := getEnv("MESSAGE_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback' for empty env var, got '%s'", result)
	}
}
