package chats

import (
	"context"
	"time"
	"github.com/google/uuid"
)

type Service interface {
	CreateChat(ctx context.Context, creatorID string, req CreateChatRequest) (Chat, error)
	GetMyChats(ctx context.Context, userID string) ([]Chat, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CreateChat(ctx context.Context, creatorID string, req CreateChatRequest) (Chat, error) {
	// Logic to ensure the creator is part of the participants
	participants := req.Participants
	participants = append(participants, creatorID)

	chat := Chat{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Type:         req.Type,
		Participants: participants,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.Create(ctx, chat); err != nil {
		return Chat{}, err
	}
	return chat, nil
}

func (s *service) GetMyChats(ctx context.Context, userID string) ([]Chat, error) {
	return s.repo.FindByUser(ctx, userID)
}