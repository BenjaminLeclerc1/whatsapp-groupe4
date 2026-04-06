package messages

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Service interface {
	SendMessage(ctx context.Context, userID string, req SendMessageRequest) (Message, error)
	GetMessage(ctx context.Context, userID, messageID string) (Message, error)
	GetMessageHistory(ctx context.Context, userID, chatID, cursor string, limit int) (MessageListResponse, error)
	DeleteMessage(ctx context.Context, userID, messageID string) error
}

type messageService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &messageService{repo: repo}
}

// SendMessage creates a message in a single DB round-trip via CTE that checks membership.
func (s *messageService) SendMessage(ctx context.Context, userID string, req SendMessageRequest) (Message, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return Message{}, fmt.Errorf("validation: content must not be empty")
	}

	msg, err := s.repo.CreateMessage(ctx, req.ChatID, userID, content)
	if err == pgx.ErrNoRows {
		return Message{}, fmt.Errorf("forbidden: you are not a member of this chat")
	}
	if err != nil {
		return Message{}, fmt.Errorf("failed to send message: %w", err)
	}
	return msg, nil
}

func (s *messageService) GetMessage(ctx context.Context, userID, messageID string) (Message, error) {
	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return Message{}, err
	}

	if err := s.requireMember(ctx, msg.ChatID, userID); err != nil {
		return Message{}, err
	}

	return msg, nil
}

func (s *messageService) GetMessageHistory(ctx context.Context, userID, chatID, cursor string, limit int) (MessageListResponse, error) {
	if err := s.requireMember(ctx, chatID, userID); err != nil {
		return MessageListResponse{}, err
	}

	msgs, err := s.repo.ListMessagesByChat(ctx, chatID, cursor, limit)
	if err != nil {
		return MessageListResponse{}, fmt.Errorf("failed to list messages: %w", err)
	}

	nextCursor := ""
	if len(msgs) > 0 {
		nextCursor = msgs[len(msgs)-1].ID
	}

	return MessageListResponse{
		Messages: msgs,
		ChatID:   chatID,
		Count:    len(msgs),
		Cursor:   nextCursor,
	}, nil
}

// DeleteMessage ensures only the sender can delete their own message.
func (s *messageService) DeleteMessage(ctx context.Context, userID, messageID string) error {
	err := s.repo.DeleteMessage(ctx, messageID, userID)
	if err == nil {
		return nil
	}

	if err.Error() == "delete_no_rows" {
		exists, existsErr := s.repo.MessageExists(ctx, messageID)
		if existsErr != nil {
			return fmt.Errorf("failed to delete message: %w", existsErr)
		}
		if !exists {
			return fmt.Errorf("message not found")
		}
		return fmt.Errorf("forbidden: you can only delete your own messages")
	}

	return fmt.Errorf("failed to delete message: %w", err)
}

func (s *messageService) requireMember(ctx context.Context, chatID, userID string) error {
	ok, err := s.repo.IsChatMember(ctx, chatID, userID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !ok {
		return fmt.Errorf("forbidden: you are not a member of this chat")
	}
	return nil
}
