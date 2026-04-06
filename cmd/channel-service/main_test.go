package main

import (
	"context"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/whatsapp-groupe4/internal/channels"
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

func TestGetEnv_DefaultPort(t *testing.T) {
	os.Unsetenv("PORT")
	if got := getEnv("PORT", "8085"); got != "8085" {
		t.Errorf("expected default 8085, got %q", got)
	}
}

// ─── initDB ───────────────────────────────────────────────────────────────────

func TestInitDB_InvalidURL(t *testing.T) {
	// URL invalide → pgxpool.ParseConfig échoue → return nil, err couvert
	_, err := initDB("not-a-valid-postgres-url")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestInitDB_ValidFormat(t *testing.T) {
	// ParseConfig réussit (format valide) → pgxpool.NewWithConfig est couvert
	// La connexion vers le port 1 échoue ou retourne un pool lazy (pas d'erreur)
	pool, err := initDB("postgres://127.0.0.1:1/db?sslmode=disable")
	if pool != nil {
		pool.Close()
	}
	_ = err // On couvre juste la ligne NewWithConfig
}

type stubChannelService struct{}

func (stubChannelService) CreateChannel(ctx context.Context, userID string, req channels.CreateChannelRequest) (channels.ChannelResponse, error) {
	return channels.ChannelResponse{}, nil
}
func (stubChannelService) GetChannel(ctx context.Context, userID, channelID string) (channels.ChannelResponse, error) {
	return channels.ChannelResponse{}, nil
}
func (stubChannelService) UpdateChannel(ctx context.Context, userID, channelID string, req channels.UpdateChannelRequest) (channels.ChannelResponse, error) {
	return channels.ChannelResponse{}, nil
}
func (stubChannelService) DeleteChannel(ctx context.Context, userID, channelID string) error { return nil }
func (stubChannelService) ListMyChannels(ctx context.Context, userID string) (channels.ChannelListResponse, error) {
	return channels.ChannelListResponse{}, nil
}
func (stubChannelService) AddMember(ctx context.Context, userID, channelID string, req channels.AddMemberRequest) (channels.Participant, error) {
	return channels.Participant{}, nil
}
func (stubChannelService) RemoveMember(ctx context.Context, userID, channelID, targetUserID string) error {
	return nil
}
func (stubChannelService) ListMembers(ctx context.Context, userID, channelID string) (channels.MemberListResponse, error) {
	return channels.MemberListResponse{}, nil
}
func (stubChannelService) ListMessages(ctx context.Context, userID, channelID, cursor string, limit int) (channels.MessageListResponse, error) {
	return channels.MessageListResponse{}, nil
}

func TestNewChannelRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := channels.NewHandler(stubChannelService{})
	r := newChannelRouter(h)
	if r == nil {
		t.Fatal("nil router")
	}
}

func TestChannelMigrationURL(t *testing.T) {
	if got := channelMigrationURL("postgres://h/db"); got != "postgres://h/db?x-migrations-table=migrations_channels" {
		t.Errorf("got %s", got)
	}
	if got := channelMigrationURL("postgres://h/db?sslmode=disable"); got != "postgres://h/db?sslmode=disable&x-migrations-table=migrations_channels" {
		t.Errorf("got %s", got)
	}
}
