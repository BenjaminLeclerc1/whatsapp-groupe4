package messages

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

// ─── Mock Repository ──────────────────────────────────────────────────────────

type mockRepo struct {
	createFn     func(ctx context.Context, chatID, senderID, content string) (Message, error)
	getByIDFn    func(ctx context.Context, id string) (Message, error)
	listFn       func(ctx context.Context, chatID, cursor string, limit int) ([]Message, error)
	deleteFn     func(ctx context.Context, id, senderID string) error
	existsFn     func(ctx context.Context, id string) (bool, error)
	isMemberFn   func(ctx context.Context, chatID, userID string) (bool, error)
}

func (m *mockRepo) CreateMessage(ctx context.Context, chatID, senderID, content string) (Message, error) {
	return m.createFn(ctx, chatID, senderID, content)
}
func (m *mockRepo) GetMessageByID(ctx context.Context, id string) (Message, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockRepo) ListMessagesByChat(ctx context.Context, chatID, cursor string, limit int) ([]Message, error) {
	return m.listFn(ctx, chatID, cursor, limit)
}
func (m *mockRepo) DeleteMessage(ctx context.Context, id, senderID string) error {
	return m.deleteFn(ctx, id, senderID)
}
func (m *mockRepo) MessageExists(ctx context.Context, id string) (bool, error) {
	return m.existsFn(ctx, id)
}
func (m *mockRepo) IsChatMember(ctx context.Context, chatID, userID string) (bool, error) {
	return m.isMemberFn(ctx, chatID, userID)
}

// ─── SendMessage ──────────────────────────────────────────────────────────────

func TestSendMessage_EmptyContent(t *testing.T) {
	svc := NewService(&mockRepo{})
	_, err := svc.SendMessage(context.Background(), "user-1", SendMessageRequest{
		ChatID:  "chat-1",
		Content: "   ",
	})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestSendMessage_Success(t *testing.T) {
	expected := Message{ID: "msg-1", Content: "hello", ChatID: "chat-1", SenderID: "user-1"}
	repo := &mockRepo{
		createFn: func(_ context.Context, chatID, senderID, content string) (Message, error) {
			return expected, nil
		},
	}
	svc := NewService(repo)
	msg, err := svc.SendMessage(context.Background(), "user-1", SendMessageRequest{
		ChatID:  "chat-1",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != "msg-1" {
		t.Errorf("expected msg-1, got %s", msg.ID)
	}
}

func TestSendMessage_NotMember(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _, _, _ string) (Message, error) {
			return Message{}, pgx.ErrNoRows
		},
	}
	svc := NewService(repo)
	_, err := svc.SendMessage(context.Background(), "user-1", SendMessageRequest{
		ChatID:  "chat-1",
		Content: "hello",
	})
	if err == nil {
		t.Error("expected error for non-member")
	}
}

func TestSendMessage_RepoError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(_ context.Context, _, _, _ string) (Message, error) {
			return Message{}, errors.New("db error")
		},
	}
	svc := NewService(repo)
	_, err := svc.SendMessage(context.Background(), "user-1", SendMessageRequest{
		ChatID:  "chat-1",
		Content: "hello",
	})
	if err == nil {
		t.Error("expected error from repo")
	}
}

// ─── GetMessage ───────────────────────────────────────────────────────────────

func TestGetMessage_Success(t *testing.T) {
	msg := Message{ID: "m1", ChatID: "c1", SenderID: "u1", Content: "test"}
	repo := &mockRepo{
		getByIDFn:  func(_ context.Context, id string) (Message, error) { return msg, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)
	result, err := svc.GetMessage(context.Background(), "u1", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "m1" {
		t.Errorf("expected m1, got %s", result.ID)
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, id string) (Message, error) {
			return Message{}, errors.New("message not found")
		},
	}
	svc := NewService(repo)
	_, err := svc.GetMessage(context.Background(), "u1", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent message")
	}
}

func TestGetMessage_NotMember(t *testing.T) {
	msg := Message{ID: "m1", ChatID: "c1"}
	repo := &mockRepo{
		getByIDFn:  func(_ context.Context, _ string) (Message, error) { return msg, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := NewService(repo)
	_, err := svc.GetMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected forbidden error for non-member")
	}
}

func TestGetMessage_MemberCheckError(t *testing.T) {
	msg := Message{ID: "m1", ChatID: "c1"}
	repo := &mockRepo{
		getByIDFn:  func(_ context.Context, _ string) (Message, error) { return msg, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, errors.New("db error") },
	}
	svc := NewService(repo)
	_, err := svc.GetMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected error from membership check")
	}
}

// ─── GetMessageHistory ────────────────────────────────────────────────────────

func TestGetMessageHistory_NotMember(t *testing.T) {
	repo := &mockRepo{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := NewService(repo)
	_, err := svc.GetMessageHistory(context.Background(), "u1", "c1", "", 10)
	if err == nil {
		t.Error("expected forbidden error")
	}
}

func TestGetMessageHistory_Success(t *testing.T) {
	msgs := []Message{
		{ID: "m1", Content: "hello"},
		{ID: "m2", Content: "world"},
	}
	repo := &mockRepo{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listFn:     func(_ context.Context, _, _ string, _ int) ([]Message, error) { return msgs, nil },
	}
	svc := NewService(repo)
	result, err := svc.GetMessageHistory(context.Background(), "u1", "c1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 2 {
		t.Errorf("expected 2 messages, got %d", result.Count)
	}
	if result.Cursor != "m2" {
		t.Errorf("expected cursor 'm2', got '%s'", result.Cursor)
	}
}

func TestGetMessageHistory_Empty(t *testing.T) {
	repo := &mockRepo{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listFn:     func(_ context.Context, _, _ string, _ int) ([]Message, error) { return []Message{}, nil },
	}
	svc := NewService(repo)
	result, err := svc.GetMessageHistory(context.Background(), "u1", "c1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Count != 0 {
		t.Errorf("expected 0 messages, got %d", result.Count)
	}
	if result.Cursor != "" {
		t.Errorf("expected empty cursor for empty result, got '%s'", result.Cursor)
	}
}

func TestGetMessageHistory_RepoError(t *testing.T) {
	repo := &mockRepo{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listFn:     func(_ context.Context, _, _ string, _ int) ([]Message, error) { return nil, errors.New("db error") },
	}
	svc := NewService(repo)
	_, err := svc.GetMessageHistory(context.Background(), "u1", "c1", "", 10)
	if err == nil {
		t.Error("expected error from repo")
	}
}

// ─── DeleteMessage ────────────────────────────────────────────────────────────

func TestDeleteMessage_Success(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error { return nil },
	}
	svc := NewService(repo)
	err := svc.DeleteMessage(context.Background(), "u1", "m1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteMessage_NotFound(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error {
			return errors.New("delete_no_rows")
		},
		existsFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}
	svc := NewService(repo)
	err := svc.DeleteMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected 'message not found' error")
	}
}

func TestDeleteMessage_Forbidden(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error {
			return errors.New("delete_no_rows")
		},
		existsFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := NewService(repo)
	err := svc.DeleteMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected forbidden error")
	}
}

func TestDeleteMessage_ExistsCheckError(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error {
			return errors.New("delete_no_rows")
		},
		existsFn: func(_ context.Context, _ string) (bool, error) { return false, errors.New("db error") },
	}
	svc := NewService(repo)
	err := svc.DeleteMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected error from exists check")
	}
}

func TestDeleteMessage_RepoError(t *testing.T) {
	repo := &mockRepo{
		deleteFn: func(_ context.Context, _, _ string) error {
			return errors.New("connection error")
		},
	}
	svc := NewService(repo)
	err := svc.DeleteMessage(context.Background(), "u1", "m1")
	if err == nil {
		t.Error("expected error from repo")
	}
}

// ─── Model tests ──────────────────────────────────────────────────────────────

func TestSendMessageRequest_Validation(t *testing.T) {
	req := SendMessageRequest{ChatID: "chat-1", Content: "hello"}
	if req.ChatID != "chat-1" {
		t.Errorf("unexpected ChatID: %s", req.ChatID)
	}
	if req.Content != "hello" {
		t.Errorf("unexpected Content: %s", req.Content)
	}
}

func TestMessageListResponse_Fields(t *testing.T) {
	resp := MessageListResponse{
		Messages: []Message{{ID: "m1"}},
		ChatID:   "c1",
		Count:    1,
		Cursor:   "m1",
	}
	if resp.Count != 1 {
		t.Errorf("expected count 1, got %d", resp.Count)
	}
}
