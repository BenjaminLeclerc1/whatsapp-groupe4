package channels

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ─── Mock Repository ──────────────────────────────────────────────────────────

type mockRepo struct {
	createChannelFn    func(ctx context.Context, ch Channel) (Channel, error)
	getChannelByIDFn   func(ctx context.Context, id string) (Channel, error)
	updateChannelFn    func(ctx context.Context, id string, req UpdateChannelRequest) (Channel, error)
	deleteChannelFn    func(ctx context.Context, id string) error
	listByUserFn       func(ctx context.Context, userID string) ([]ChannelResponse, error)
	addMemberFn        func(ctx context.Context, chatID, userID string) (Participant, error)
	removeMemberFn     func(ctx context.Context, chatID, userID string) error
	isMemberFn         func(ctx context.Context, chatID, userID string) (bool, error)
	listMembersFn      func(ctx context.Context, chatID string) ([]Participant, error)
	countMembersFn     func(ctx context.Context, chatID string) (int, error)
	listMessagesFn     func(ctx context.Context, chatID, cursor string, limit int) ([]Message, error)
}

func (m *mockRepo) CreateChannel(ctx context.Context, ch Channel) (Channel, error) {
	return m.createChannelFn(ctx, ch)
}
func (m *mockRepo) GetChannelByID(ctx context.Context, id string) (Channel, error) {
	return m.getChannelByIDFn(ctx, id)
}
func (m *mockRepo) UpdateChannel(ctx context.Context, id string, req UpdateChannelRequest) (Channel, error) {
	return m.updateChannelFn(ctx, id, req)
}
func (m *mockRepo) DeleteChannel(ctx context.Context, id string) error {
	return m.deleteChannelFn(ctx, id)
}
func (m *mockRepo) ListChannelsByUser(ctx context.Context, userID string) ([]ChannelResponse, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockRepo) AddMember(ctx context.Context, chatID, userID string) (Participant, error) {
	return m.addMemberFn(ctx, chatID, userID)
}
func (m *mockRepo) RemoveMember(ctx context.Context, chatID, userID string) error {
	return m.removeMemberFn(ctx, chatID, userID)
}
func (m *mockRepo) IsMember(ctx context.Context, chatID, userID string) (bool, error) {
	return m.isMemberFn(ctx, chatID, userID)
}
func (m *mockRepo) ListMembers(ctx context.Context, chatID string) ([]Participant, error) {
	return m.listMembersFn(ctx, chatID)
}
func (m *mockRepo) CountMembers(ctx context.Context, chatID string) (int, error) {
	return m.countMembersFn(ctx, chatID)
}
func (m *mockRepo) ListMessages(ctx context.Context, chatID, cursor string, limit int) ([]Message, error) {
	return m.listMessagesFn(ctx, chatID, cursor, limit)
}

// ─── CreateChannel ────────────────────────────────────────────────────────────

func TestCreateChannel_Success(t *testing.T) {
	created := Channel{ID: "ch-1", Name: "Test", OwnerID: "u1", MaxMembers: 100}
	repo := &mockRepo{
		createChannelFn: func(_ context.Context, _ Channel) (Channel, error) { return created, nil },
		addMemberFn:     func(_ context.Context, _, _ string) (Participant, error) { return Participant{}, nil },
	}
	svc := NewService(repo)

	resp, err := svc.CreateChannel(context.Background(), "u1", CreateChannelRequest{
		Name:       "Test",
		MaxMembers: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Channel.ID != "ch-1" {
		t.Errorf("expected ch-1, got %s", resp.Channel.ID)
	}
	if resp.MemberCount != 1 {
		t.Errorf("expected 1 member, got %d", resp.MemberCount)
	}
}

func TestCreateChannel_DefaultMaxMembers(t *testing.T) {
	var gotCh Channel
	repo := &mockRepo{
		createChannelFn: func(_ context.Context, ch Channel) (Channel, error) {
			gotCh = ch
			return ch, nil
		},
		addMemberFn: func(_ context.Context, _, _ string) (Participant, error) { return Participant{}, nil },
	}
	svc := NewService(repo)

	svc.CreateChannel(context.Background(), "u1", CreateChannelRequest{Name: "Test"})
	if gotCh.MaxMembers != 1000 {
		t.Errorf("expected default 1000 max members, got %d", gotCh.MaxMembers)
	}
}

func TestCreateChannel_RepoError(t *testing.T) {
	repo := &mockRepo{
		createChannelFn: func(_ context.Context, _ Channel) (Channel, error) {
			return Channel{}, errors.New("db error")
		},
	}
	svc := NewService(repo)
	_, err := svc.CreateChannel(context.Background(), "u1", CreateChannelRequest{Name: "Test"})
	if err == nil {
		t.Error("expected error from repo")
	}
}

func TestCreateChannel_AddMemberError(t *testing.T) {
	repo := &mockRepo{
		createChannelFn: func(_ context.Context, ch Channel) (Channel, error) { return ch, nil },
		addMemberFn:     func(_ context.Context, _, _ string) (Participant, error) { return Participant{}, errors.New("add error") },
	}
	svc := NewService(repo)
	_, err := svc.CreateChannel(context.Background(), "u1", CreateChannelRequest{Name: "Test"})
	if err == nil {
		t.Error("expected error when adding owner fails")
	}
}

// ─── GetChannel ───────────────────────────────────────────────────────────────

func TestGetChannel_Success(t *testing.T) {
	ch := Channel{ID: "ch-1", Name: "Test"}
	repo := &mockRepo{
		isMemberFn:     func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		countMembersFn: func(_ context.Context, _ string) (int, error) { return 5, nil },
	}
	svc := NewService(repo)
	resp, err := svc.GetChannel(context.Background(), "u1", "ch-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MemberCount != 5 {
		t.Errorf("expected 5 members, got %d", resp.MemberCount)
	}
}

func TestGetChannel_NotMember(t *testing.T) {
	repo := &mockRepo{
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
	svc := NewService(repo)
	_, err := svc.GetChannel(context.Background(), "u1", "ch-1")
	if err == nil {
		t.Error("expected forbidden error")
	}
}

// ─── UpdateChannel ────────────────────────────────────────────────────────────

func TestUpdateChannel_Success(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "u1", Name: "Updated"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		updateChannelFn:  func(_ context.Context, _ string, _ UpdateChannelRequest) (Channel, error) { return ch, nil },
		countMembersFn:   func(_ context.Context, _ string) (int, error) { return 3, nil },
	}
	svc := NewService(repo)
	name := "Updated"
	resp, err := svc.UpdateChannel(context.Background(), "u1", "ch-1", UpdateChannelRequest{Name: &name})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Channel.ID != "ch-1" {
		t.Errorf("expected ch-1, got %s", resp.Channel.ID)
	}
}

func TestUpdateChannel_NotOwner(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "owner-1"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
	}
	svc := NewService(repo)
	_, err := svc.UpdateChannel(context.Background(), "not-owner", "ch-1", UpdateChannelRequest{})
	if err == nil {
		t.Error("expected forbidden error for non-owner")
	}
}

// ─── DeleteChannel ────────────────────────────────────────────────────────────

func TestDeleteChannel_Success(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "u1"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		deleteChannelFn:  func(_ context.Context, _ string) error { return nil },
	}
	svc := NewService(repo)
	err := svc.DeleteChannel(context.Background(), "u1", "ch-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteChannel_NotOwner(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "real-owner"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
	}
	svc := NewService(repo)
	err := svc.DeleteChannel(context.Background(), "not-owner", "ch-1")
	if err == nil {
		t.Error("expected forbidden error")
	}
}

// ─── ListMyChannels ───────────────────────────────────────────────────────────

func TestListMyChannels_Success(t *testing.T) {
	channels := []ChannelResponse{
		{Channel: Channel{ID: "c1"}, MemberCount: 3},
		{Channel: Channel{ID: "c2"}, MemberCount: 5},
	}
	repo := &mockRepo{
		listByUserFn: func(_ context.Context, _ string) ([]ChannelResponse, error) { return channels, nil },
	}
	svc := NewService(repo)
	resp, err := svc.ListMyChannels(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected count 2, got %d", resp.Count)
	}
}

// ─── AddMember ────────────────────────────────────────────────────────────────

func TestAddMember_Success(t *testing.T) {
	ch := Channel{ID: "ch-1", MaxMembers: 100}
	repo := &mockRepo{
		isMemberFn:       func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		countMembersFn:   func(_ context.Context, _ string) (int, error) { return 5, nil },
		addMemberFn:      func(_ context.Context, _, _ string) (Participant, error) { return Participant{UserID: "new-user"}, nil },
	}
	svc := NewService(repo)
	p, err := svc.AddMember(context.Background(), "u1", "ch-1", AddMemberRequest{UserID: "new-user"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.UserID != "new-user" {
		t.Errorf("expected new-user, got %s", p.UserID)
	}
}

func TestAddMember_MaxReached(t *testing.T) {
	ch := Channel{ID: "ch-1", MaxMembers: 3}
	repo := &mockRepo{
		isMemberFn:       func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		countMembersFn:   func(_ context.Context, _ string) (int, error) { return 3, nil },
	}
	svc := NewService(repo)
	_, err := svc.AddMember(context.Background(), "u1", "ch-1", AddMemberRequest{UserID: "new-user"})
	if err == nil {
		t.Error("expected max members error")
	}
}

// ─── RemoveMember ─────────────────────────────────────────────────────────────

func TestRemoveMember_SelfRemove(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "owner"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		removeMemberFn:   func(_ context.Context, _, _ string) error { return nil },
	}
	svc := NewService(repo)
	err := svc.RemoveMember(context.Background(), "member-1", "ch-1", "member-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveMember_OwnerRemovesOther(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "owner"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
		removeMemberFn:   func(_ context.Context, _, _ string) error { return nil },
	}
	svc := NewService(repo)
	err := svc.RemoveMember(context.Background(), "owner", "ch-1", "member-1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRemoveMember_OwnerCantLeaveSelf(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "owner"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
	}
	svc := NewService(repo)
	err := svc.RemoveMember(context.Background(), "owner", "ch-1", "owner")
	if err == nil {
		t.Error("expected error: owner cannot leave their own channel")
	}
}

func TestRemoveMember_Forbidden(t *testing.T) {
	ch := Channel{ID: "ch-1", OwnerID: "owner"}
	repo := &mockRepo{
		getChannelByIDFn: func(_ context.Context, _ string) (Channel, error) { return ch, nil },
	}
	svc := NewService(repo)
	err := svc.RemoveMember(context.Background(), "random-user", "ch-1", "another-user")
	if err == nil {
		t.Error("expected forbidden error")
	}
}

// ─── ListMembers ──────────────────────────────────────────────────────────────

func TestListMembers_Success(t *testing.T) {
	members := []Participant{
		{UserID: "u1", ChatID: "ch-1"},
		{UserID: "u2", ChatID: "ch-1"},
	}
	repo := &mockRepo{
		isMemberFn:    func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listMembersFn: func(_ context.Context, _ string) ([]Participant, error) { return members, nil },
	}
	svc := NewService(repo)
	resp, err := svc.ListMembers(context.Background(), "u1", "ch-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected 2 members, got %d", resp.Count)
	}
}

// ─── ListMessages ─────────────────────────────────────────────────────────────

func TestListMessages_Success(t *testing.T) {
	msgs := []Message{
		{ID: "m1", Content: "hello"},
		{ID: "m2", Content: "world"},
	}
	repo := &mockRepo{
		isMemberFn:     func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listMessagesFn: func(_ context.Context, _, _ string, _ int) ([]Message, error) { return msgs, nil },
	}
	svc := NewService(repo)
	resp, err := svc.ListMessages(context.Background(), "u1", "ch-1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("expected 2 messages, got %d", resp.Count)
	}
	if resp.Cursor != "m2" {
		t.Errorf("expected cursor 'm2', got '%s'", resp.Cursor)
	}
}

func TestListMessages_Empty(t *testing.T) {
	repo := &mockRepo{
		isMemberFn:     func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		listMessagesFn: func(_ context.Context, _, _ string, _ int) ([]Message, error) { return []Message{}, nil },
	}
	svc := NewService(repo)
	resp, err := svc.ListMessages(context.Background(), "u1", "ch-1", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Cursor != "" {
		t.Errorf("expected empty cursor, got '%s'", resp.Cursor)
	}
}

// ─── Model tests ──────────────────────────────────────────────────────────────

func TestChannelModel(t *testing.T) {
	now := time.Now()
	ch := Channel{
		ID:          "ch-1",
		Name:        "Test Channel",
		Description: "A test channel",
		IsGroup:     true,
		OwnerID:     "user-1",
		MaxMembers:  500,
		CreatedAt:   now,
	}
	if ch.MaxMembers != 500 {
		t.Errorf("unexpected MaxMembers: %d", ch.MaxMembers)
	}
	if !ch.IsGroup {
		t.Error("expected IsGroup to be true")
	}
}

func TestCreateChannelRequest_Fields(t *testing.T) {
	req := CreateChannelRequest{
		Name:        "My Channel",
		Description: "A description",
		IsGroup:     true,
		MaxMembers:  100,
	}
	if req.Name != "My Channel" {
		t.Errorf("unexpected name: %s", req.Name)
	}
}

func TestAddMemberRequest_Fields(t *testing.T) {
	req := AddMemberRequest{UserID: "user-123"}
	if req.UserID != "user-123" {
		t.Errorf("unexpected UserID: %s", req.UserID)
	}
}
