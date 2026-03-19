package channels

import (
	"context"
	"fmt"
)

type Service interface {
	CreateChannel(ctx context.Context, userID string, req CreateChannelRequest) (ChannelResponse, error)
	GetChannel(ctx context.Context, userID, channelID string) (ChannelResponse, error)
	UpdateChannel(ctx context.Context, userID, channelID string, req UpdateChannelRequest) (ChannelResponse, error)
	DeleteChannel(ctx context.Context, userID, channelID string) error
	ListMyChannels(ctx context.Context, userID string) (ChannelListResponse, error)

	AddMember(ctx context.Context, userID, channelID string, req AddMemberRequest) (Participant, error)
	RemoveMember(ctx context.Context, userID, channelID, targetUserID string) error
	ListMembers(ctx context.Context, userID, channelID string) (MemberListResponse, error)

	ListMessages(ctx context.Context, userID, channelID, cursor string, limit int) (MessageListResponse, error)
}

type channelService struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &channelService{repo: repo}
}

func (s *channelService) CreateChannel(ctx context.Context, userID string, req CreateChannelRequest) (ChannelResponse, error) {
	maxMembers := req.MaxMembers
	if maxMembers == 0 {
		maxMembers = 1000
	}

	ch := Channel{
		Name:        req.Name,
		Description: req.Description,
		IsGroup:     req.IsGroup,
		OwnerID:     userID,
		MaxMembers:  maxMembers,
	}

	created, err := s.repo.CreateChannel(ctx, ch)
	if err != nil {
		return ChannelResponse{}, fmt.Errorf("failed to create channel: %w", err)
	}

	// Owner auto-joins the channel
	if _, err := s.repo.AddMember(ctx, created.ID, userID); err != nil {
		return ChannelResponse{}, fmt.Errorf("failed to add owner as member: %w", err)
	}

	return ChannelResponse{Channel: created, MemberCount: 1}, nil
}

func (s *channelService) GetChannel(ctx context.Context, userID, channelID string) (ChannelResponse, error) {
	if err := s.requireMember(ctx, channelID, userID); err != nil {
		return ChannelResponse{}, err
	}

	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return ChannelResponse{}, err
	}

	count, err := s.repo.CountMembers(ctx, channelID)
	if err != nil {
		return ChannelResponse{}, err
	}

	return ChannelResponse{Channel: ch, MemberCount: count}, nil
}

func (s *channelService) UpdateChannel(ctx context.Context, userID, channelID string, req UpdateChannelRequest) (ChannelResponse, error) {
	if err := s.requireOwner(ctx, channelID, userID); err != nil {
		return ChannelResponse{}, err
	}

	ch, err := s.repo.UpdateChannel(ctx, channelID, req)
	if err != nil {
		return ChannelResponse{}, err
	}

	count, err := s.repo.CountMembers(ctx, channelID)
	if err != nil {
		return ChannelResponse{}, err
	}

	return ChannelResponse{Channel: ch, MemberCount: count}, nil
}

func (s *channelService) DeleteChannel(ctx context.Context, userID, channelID string) error {
	if err := s.requireOwner(ctx, channelID, userID); err != nil {
		return err
	}
	return s.repo.DeleteChannel(ctx, channelID)
}

func (s *channelService) ListMyChannels(ctx context.Context, userID string) (ChannelListResponse, error) {
	channels, err := s.repo.ListChannelsByUser(ctx, userID)
	if err != nil {
		return ChannelListResponse{}, err
	}
	return ChannelListResponse{Channels: channels, Count: len(channels)}, nil
}

// --- Members ---

func (s *channelService) AddMember(ctx context.Context, userID, channelID string, req AddMemberRequest) (Participant, error) {
	if err := s.requireMember(ctx, channelID, userID); err != nil {
		return Participant{}, err
	}

	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return Participant{}, err
	}

	count, err := s.repo.CountMembers(ctx, channelID)
	if err != nil {
		return Participant{}, err
	}
	if count >= ch.MaxMembers {
		return Participant{}, fmt.Errorf("channel has reached the maximum number of members (%d)", ch.MaxMembers)
	}

	p, err := s.repo.AddMember(ctx, channelID, req.UserID)
	if err != nil {
		return Participant{}, err
	}
	return p, nil
}

func (s *channelService) RemoveMember(ctx context.Context, userID, channelID, targetUserID string) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return err
	}

	isOwner := ch.OwnerID == userID
	isSelf := userID == targetUserID

	if !isOwner && !isSelf {
		return fmt.Errorf("forbidden: only the owner or the member themselves can remove a member")
	}

	if isOwner && isSelf {
		return fmt.Errorf("forbidden: the owner cannot leave the channel, transfer ownership or delete it instead")
	}

	return s.repo.RemoveMember(ctx, channelID, targetUserID)
}

func (s *channelService) ListMembers(ctx context.Context, userID, channelID string) (MemberListResponse, error) {
	if err := s.requireMember(ctx, channelID, userID); err != nil {
		return MemberListResponse{}, err
	}

	members, err := s.repo.ListMembers(ctx, channelID)
	if err != nil {
		return MemberListResponse{}, err
	}
	return MemberListResponse{Members: members, Count: len(members)}, nil
}

// --- Messages ---

func (s *channelService) ListMessages(ctx context.Context, userID, channelID, cursor string, limit int) (MessageListResponse, error) {
	if err := s.requireMember(ctx, channelID, userID); err != nil {
		return MessageListResponse{}, err
	}

	msgs, err := s.repo.ListMessages(ctx, channelID, cursor, limit)
	if err != nil {
		return MessageListResponse{}, err
	}

	nextCursor := ""
	if len(msgs) > 0 {
		nextCursor = msgs[len(msgs)-1].ID
	}

	return MessageListResponse{
		Messages: msgs,
		ChatID:   channelID,
		Count:    len(msgs),
		Cursor:   nextCursor,
	}, nil
}

// --- Authorization helpers ---

func (s *channelService) requireMember(ctx context.Context, channelID, userID string) error {
	ok, err := s.repo.IsMember(ctx, channelID, userID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !ok {
		return fmt.Errorf("forbidden: you are not a member of this channel")
	}
	return nil
}

func (s *channelService) requireOwner(ctx context.Context, channelID, userID string) error {
	ch, err := s.repo.GetChannelByID(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.OwnerID != userID {
		return fmt.Errorf("forbidden: only the channel owner can perform this action")
	}
	return nil
}
