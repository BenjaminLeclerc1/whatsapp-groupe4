package main

import (
	"context"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/whatsapp-groupe4/internal/chats"
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

func TestChatMigrationURL(t *testing.T) {
	if got := chatMigrationURL("postgres://user:pass@localhost/db"); got != "postgres://user:pass@localhost/db?x-migrations-table=migrations_chats" {
		t.Errorf("unexpected: %s", got)
	}
	if got := chatMigrationURL("postgres://user:pass@localhost/db?sslmode=disable"); got != "postgres://user:pass@localhost/db?sslmode=disable&x-migrations-table=migrations_chats" {
		t.Errorf("unexpected: %s", got)
	}
}

type stubChatService struct{}

func (stubChatService) CreateChat(ctx context.Context, creatorID string, req chats.CreateChatRequest) (chats.Chat, error) {
	return chats.Chat{}, nil
}
func (stubChatService) GetMyChats(ctx context.Context, userID string) ([]chats.Chat, error) {
	return nil, nil
}

func TestNewChatRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := chats.NewHandler(stubChatService{})
	r := newChatRouter(h)
	if r == nil {
		t.Fatal("nil router")
	}
}

// ─── initDB — couvre le chemin d'erreur (URL invalide) ───────────────────────

func TestInitDB_InvalidURL(t *testing.T) {
	_, err := initDB("not-a-valid-postgres-url")
	if err == nil {
		t.Error("expected error for invalid DB URL")
	}
}
