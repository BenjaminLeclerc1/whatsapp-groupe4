package logger

import (
	"os"
	"testing"
)

func TestInitDevAndClose_CreatesDevLogFile(t *testing.T) {
	_ = os.Setenv("APP_ENV", "dev")
	defer os.Unsetenv("APP_ENV")

	Init("logger-test")
	Close()

	if _, err := os.Stat("logs/dev.log"); err != nil {
		t.Fatalf("expected logs/dev.log to exist, got error: %v", err)
	}
}

func TestInfoAndError_DoNotPanic(t *testing.T) {
	_ = os.Setenv("APP_ENV", "dev")
	defer os.Unsetenv("APP_ENV")

	Init("logger-test")
	defer Close()

	Info("hello %s", "world")
	Error("boom %d", 1)
}
