package chats

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ─── Mock Repository ──────────────────────────────────────────────────────────

type mockRepo struct {
	createFn      func(ctx context.Context, chat Chat) error
	findByUserFn  func(ctx context.Context, userID string) ([]Chat, error)
}

func (m *mockRepo) Create(ctx context.Context, chat Chat) error {
	return m.createFn(ctx, chat)
}
func (m *mockRepo) FindByUser(ctx context.Context, userID string) ([]Chat, error) {
	return m.findByUserFn(ctx, userID)
}

// ─── CreateChat ───────────────────────────────────────────────────────────────

func TestCreateChat_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ Chat) error { return nil },
	}
	svc := NewService(repo)

	chat, err := svc.CreateChat(context.Background(), "creator-1", CreateChatRequest{
		Participants: []string{"user-2"},
		Type:         "private",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chat.ID == "" {
		t.Error("expected non-empty chat ID")
	}
	if chat.Type != "private" {
		t.Errorf("expected type 'private', got '%s'", chat.Type)
	}
}

func TestCreateChat_CreatorAddedToParticipants(t *testing.T) {
	var createdChat Chat
	repo := &mockRepo{
		createFn: func(_ context.Context, chat Chat) error {
			createdChat = chat
			return nil
		},
	}
	svc := NewService(repo)

	_, err := svc.CreateChat(context.Background(), "creator-99", CreateChatRequest{
		Participants: []string{"user-1", "user-2"},
		Type:         "group",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, p := range createdChat.Participants {
		if p == "creator-99" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected creator to be in participants")
	}
	if len(createdChat.Participants) != 3 {
		t.Errorf("expected 3 participants (2 + creator), got %d", len(createdChat.Participants))
	}
}

func TestCreateChat_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ Chat) error { return errors.New("db error") },
	}
	svc := NewService(repo)

	_, err := svc.CreateChat(context.Background(), "creator-1", CreateChatRequest{
		Participants: []string{"user-2"},
		Type:         "private",
	})
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestCreateChat_GroupType(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _ Chat) error { return nil },
	}
	svc := NewService(repo)

	chat, _ := svc.CreateChat(context.Background(), "owner", CreateChatRequest{
		Participants: []string{"a", "b", "c"},
		Type:         "group",
		Name:         "Mon groupe",
	})
	if chat.Name != "Mon groupe" {
		t.Errorf("expected 'Mon groupe', got '%s'", chat.Name)
	}
}

func TestCreateChat_HasCreatedAt(t *testing.T) {
	before := time.Now()
	repo := &mockRepo{
		createFn: func(_ context.Context, _ Chat) error { return nil },
	}
	svc := NewService(repo)

	chat, _ := svc.CreateChat(context.Background(), "u1", CreateChatRequest{
		Participants: []string{"u2"},
		Type:         "private",
	})
	if chat.CreatedAt.Before(before) {
		t.Error("expected CreatedAt to be after test start")
	}
}

// ─── GetMyChats ───────────────────────────────────────────────────────────────

func TestGetMyChats_Success(t *testing.T) {
	expected := []Chat{
		{ID: "c1", Type: "private"},
		{ID: "c2", Type: "group"},
	}
	repo := &mockRepo{
		findByUserFn: func(_ context.Context, _ string) ([]Chat, error) { return expected, nil },
	}
	svc := NewService(repo)

	chats, err := svc.GetMyChats(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chats) != 2 {
		t.Errorf("expected 2 chats, got %d", len(chats))
	}
}

func TestGetMyChats_Empty(t *testing.T) {
	repo := &mockRepo{
		findByUserFn: func(_ context.Context, _ string) ([]Chat, error) { return []Chat{}, nil },
	}
	svc := NewService(repo)

	chats, err := svc.GetMyChats(context.Background(), "user-lonely")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chats) != 0 {
		t.Errorf("expected 0 chats, got %d", len(chats))
	}
}

func TestGetMyChats_RepoError(t *testing.T) {
	repo := &mockRepo{
		findByUserFn: func(_ context.Context, _ string) ([]Chat, error) {
			return nil, errors.New("db error")
		},
	}
	svc := NewService(repo)

	_, err := svc.GetMyChats(context.Background(), "user-1")
	if err == nil {
		t.Error("expected error from repo")
	}
}

// ─── Model tests ──────────────────────────────────────────────────────────────

func TestChatModel(t *testing.T) {
	chat := Chat{
		ID:           "chat-1",
		Name:         "Test",
		Type:         "group",
		Participants: []string{"u1", "u2"},
	}
	if chat.ID != "chat-1" {
		t.Errorf("unexpected ID: %s", chat.ID)
	}
	if len(chat.Participants) != 2 {
		t.Errorf("expected 2 participants, got %d", len(chat.Participants))
	}
}

func TestCreateChatRequest_Fields(t *testing.T) {
	req := CreateChatRequest{
		Participants: []string{"u1", "u2"},
		Type:         "group",
		Name:         "My Group",
	}
	if req.Type != "group" {
		t.Errorf("unexpected type: %s", req.Type)
	}
}
