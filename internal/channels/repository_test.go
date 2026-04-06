package channels

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v2"
)

type pgxMockExpectations interface {
	ExpectQuery(string) *pgxmock.ExpectedQuery
	ExpectExec(string) *pgxmock.ExpectedExec
	ExpectationsWereMet() error
}

func newChanMock(t *testing.T) (pgxPool, pgxMockExpectations, func()) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	exp, ok := mock.(pgxMockExpectations)
	if !ok {
		t.Fatal("pgxmock: pool does not expose Expect*")
	}
	return mock, exp, func() { mock.Close() }
}

func TestPgRepository_CreateChannel(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	ts := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_group", "owner_id", "max_members", "created_at"}).
		AddRow("ch1", "name", "", true, "o1", 100, ts)
	exp.ExpectQuery(`INSERT INTO chats`).
		WithArgs("name", "", true, "o1", 100).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	ch := Channel{Name: "name", Description: "", IsGroup: true, OwnerID: "o1", MaxMembers: 100}
	got, err := repo.CreateChannel(context.Background(), ch)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "ch1" {
		t.Fatalf("got id %q", got.ID)
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_GetChannelByID_NotFound(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	exp.ExpectQuery(`FROM chats WHERE id`).
		WithArgs("id1").
		WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err := repo.GetChannelByID(context.Background(), "id1")
	if err == nil || !strings.Contains(err.Error(), "channel not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestPgRepository_GetChannelByID_OK(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	ts := time.Now()
	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_group", "owner_id", "max_members", "created_at"}).
		AddRow("ch1", "n", "", true, "o1", 100, ts)
	exp.ExpectQuery(`FROM chats WHERE id`).
		WithArgs("ch1").
		WillReturnRows(rows)

	repo := NewRepository(mock)
	ch, err := repo.GetChannelByID(context.Background(), "ch1")
	if err != nil {
		t.Fatal(err)
	}
	if ch.ID != "ch1" {
		t.Fatal("bad id")
	}
}

func TestPgRepository_UpdateChannel_NotFound(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	exp.ExpectQuery(`UPDATE chats`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), "id1").
		WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err := repo.UpdateChannel(context.Background(), "id1", UpdateChannelRequest{})
	if err == nil || !strings.Contains(err.Error(), "channel not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestPgRepository_DeleteChannel_NoRows(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	exp.ExpectExec(`DELETE FROM chats WHERE id`).
		WithArgs("id1").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	repo := NewRepository(mock)
	err := repo.DeleteChannel(context.Background(), "id1")
	if err == nil || !strings.Contains(err.Error(), "channel not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestPgRepository_ListChannelsByUser(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "name", "description", "is_group", "owner_id", "max_members", "created_at", "member_count"})
	exp.ExpectQuery(`SELECT c.id, c.name`).
		WithArgs("u1").
		WillReturnRows(rows)

	repo := NewRepository(mock)
	list, err := repo.ListChannelsByUser(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("len=%d", len(list))
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_AddMember_AlreadyMember(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	exp.ExpectQuery(`INSERT INTO chat_participants`).
		WithArgs("c1", "u1").
		WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err := repo.AddMember(context.Background(), "c1", "u1")
	if err == nil || !strings.Contains(err.Error(), "already a member") {
		t.Fatalf("err=%v", err)
	}
}

func TestPgRepository_RemoveMember_NotFound(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	exp.ExpectExec(`DELETE FROM chat_participants`).
		WithArgs("c1", "u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 0))

	repo := NewRepository(mock)
	err := repo.RemoveMember(context.Background(), "c1", "u1")
	if err == nil || !strings.Contains(err.Error(), "member not found") {
		t.Fatalf("err=%v", err)
	}
}

func TestPgRepository_IsMember(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
	exp.ExpectQuery(`SELECT EXISTS`).WillReturnRows(rows)

	repo := NewRepository(mock)
	ok, err := repo.IsMember(context.Background(), "c1", "u1")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestPgRepository_CountMembers(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"count"}).AddRow(3)
	exp.ExpectQuery(`COUNT\(\*\) FROM chat_participants WHERE chat_id`).
		WithArgs("c1").
		WillReturnRows(rows)

	repo := NewRepository(mock)
	n, err := repo.CountMembers(context.Background(), "c1")
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestPgRepository_ListMembers(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"chat_id", "user_id", "joined_at"})
	exp.ExpectQuery(`SELECT chat_id, user_id, joined_at`).
		WithArgs("c1").
		WillReturnRows(rows)

	repo := NewRepository(mock)
	list, err := repo.ListMembers(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("len=%d", len(list))
	}
}

func TestPgRepository_ListMessages_NoCursor(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`FROM messages`).
		WithArgs("c1", 10).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	msgs, err := repo.ListMessages(context.Background(), "c1", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("len=%d", len(msgs))
	}
}

func TestPgRepository_ListMessages_WithCursor(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`FROM messages`).
		WithArgs("c1", "mid", 10).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	_, err := repo.ListMessages(context.Background(), "c1", "mid", 10)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_ListMessages_LimitClamp(t *testing.T) {
	mock, exp, done := newChanMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`FROM messages`).
		WithArgs("c1", 50).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	_, err := repo.ListMessages(context.Background(), "c1", "", 200)
	if err != nil {
		t.Fatal(err)
	}
}
