package postgres

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestUserRepository_CreateAndGet(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, err := repo.Create(ctx, "alice", "hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 || created.Login != "alice" {
		t.Fatalf("unexpected user: %+v", created)
	}

	byLogin, err := repo.GetByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if byLogin.ID != created.ID {
		t.Fatalf("GetByLogin id = %d, want %d", byLogin.ID, created.ID)
	}

	byID, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Login != "alice" {
		t.Fatalf("GetByID login = %q, want alice", byID.Login)
	}
}

func TestUserRepository_CreateDuplicateLogin(t *testing.T) {
	db := newTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	if _, err := repo.Create(ctx, "bob", "hash"); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err := repo.Create(ctx, "bob", "other")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second Create error = %v, want ErrConflict", err)
	}
}
