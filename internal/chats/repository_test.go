package chats

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v2"
)

func newChatsMock(t *testing.T) (pgxPool, func()) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	mock.QueryMatcher = pgxmock.QueryMatcherRegexp
	return mock, func() { mock.Close() }
}

func TestRepository_Create(t *testing.T) {
	mock, done := newChatsMock(t)
	defer done()

	mock.ExpectExec(`INSERT INTO chats`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := NewRepository(mock)
	err := repo.Create(context.Background(), Chat{
		ID: "c1", Name: "g", Type: "group", Participants: []string{"u1"}, CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_FindByUser_Empty(t *testing.T) {
	mock, done := newChatsMock(t)
	defer done()

	rows := pgxmock.NewRows([]string{"id", "name", "type", "participants", "created_at"})
	mock.ExpectQuery(`SELECT id, name, type, participants, created_at`).WillReturnRows(rows)

	repo := NewRepository(mock)
	chats, err := repo.FindByUser(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(chats) != 0 {
		t.Fatalf("expected 0 chats, got %d", len(chats))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_UpdateName(t *testing.T) {
	mock, done := newChatsMock(t)
	defer done()

	mock.ExpectExec(`UPDATE chats SET name`).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewRepository(mock)
	if err := repo.UpdateName(context.Background(), "id1", "new"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRepository_Delete(t *testing.T) {
	mock, done := newChatsMock(t)
	defer done()

	mock.ExpectExec(`DELETE FROM chats WHERE id`).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewRepository(mock)
	if err := repo.Delete(context.Background(), "id1"); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
