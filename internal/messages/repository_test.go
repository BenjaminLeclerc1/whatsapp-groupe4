package messages

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

func newMsgMock(t *testing.T) (pgxPool, pgxMockExpectations, func()) {
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

func TestPgRepository_CreateMessage(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"}).
		AddRow("m1", "s1", "c1", "hi", "sent", time.Now())
	exp.ExpectQuery(`INSERT INTO messages`).WillReturnRows(rows)

	repo := NewRepository(mock)
	m, err := repo.CreateMessage(context.Background(), "c1", "s1", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "m1" || m.Content != "hi" {
		t.Fatalf("unexpected message: %+v", m)
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_GetMessageByID_NotFound(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	exp.ExpectQuery(`SELECT id, sender_id::text`).WillReturnError(pgx.ErrNoRows)

	repo := NewRepository(mock)
	_, err := repo.GetMessageByID(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestPgRepository_ListMessagesByChat_NoCursor(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`SELECT id, sender_id::text, chat_id, content, status, created_at FROM messages WHERE chat_id`).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	msgs, err := repo.ListMessagesByChat(context.Background(), "c1", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_ListMessagesByChat_WithCursor(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`SELECT id, sender_id::text, chat_id, content, status, created_at FROM messages WHERE chat_id`).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	_, err := repo.ListMessagesByChat(context.Background(), "c1", "cursor-id", 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_ListMessagesByChat_LimitClamp(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "sender_id", "chat_id", "content", "status", "created_at"})
	exp.ExpectQuery(`SELECT id, sender_id::text, chat_id, content, status, created_at FROM messages WHERE chat_id`).
		WillReturnRows(rows)

	repo := NewRepository(mock)
	_, err := repo.ListMessagesByChat(context.Background(), "c1", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := exp.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPgRepository_DeleteMessage_NoRows(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	exp.ExpectExec(`DELETE FROM messages WHERE id`).WillReturnResult(pgxmock.NewResult("DELETE", 0))

	repo := NewRepository(mock)
	err := repo.DeleteMessage(context.Background(), "m1", "s1")
	if err == nil || !strings.Contains(err.Error(), "delete_failed") {
		t.Fatalf("expected delete failed error, got %v", err)
	}
}

func TestPgRepository_MessageExists(t *testing.T) {
	mock, exp, done := newMsgMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
	exp.ExpectQuery(`SELECT EXISTS`).WillReturnRows(rows)

	repo := NewRepository(mock)
	ok, err := repo.MessageExists(context.Background(), "m1")
	if err != nil || !ok {
		t.Fatalf("exists=%v err=%v", ok, err)
	}
}

func TestPgRepository_IsChatMember(t *testing.T) {
	mock, _, done := newMsgMock(t)
	defer done()

	repo := NewRepository(mock)
	ok, err := repo.IsChatMember(context.Background(), "c1", "u1")
	if err != nil || !ok {
		t.Fatalf("expected true, nil; got %v %v", ok, err)
	}
}
