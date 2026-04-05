package main

import (
	"os"
	"testing"
)

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("CHANNEL_TEST_NONEXISTENT", "default")
	if result != "default" {
		t.Errorf("expected 'default', got '%s'", result)
	}
}

func TestGetEnv_Set(t *testing.T) {
	os.Setenv("CHANNEL_TEST_VAR", "set_value")
	defer os.Unsetenv("CHANNEL_TEST_VAR")

	result := getEnv("CHANNEL_TEST_VAR", "default")
	if result != "set_value" {
		t.Errorf("expected 'set_value', got '%s'", result)
	}
}

func TestGetEnv_EmptyFallsBack(t *testing.T) {
	os.Setenv("CHANNEL_EMPTY_VAR", "")
	defer os.Unsetenv("CHANNEL_EMPTY_VAR")

	result := getEnv("CHANNEL_EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("expected 'fallback', got '%s'", result)
	}
}

func TestRequireEnv_Set(t *testing.T) {
	os.Setenv("CHANNEL_REQUIRED_VAR", "present")
	defer os.Unsetenv("CHANNEL_REQUIRED_VAR")

	result := requireEnv("CHANNEL_REQUIRED_VAR")
	if result != "present" {
		t.Errorf("expected 'present', got '%s'", result)
	}
}
