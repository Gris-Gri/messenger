package postgres

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
)

func TestMemberRepository_AddGetRemove(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	memberRepo := NewMemberRepository(db)
	ctx := context.Background()

	adminID := createTestUser(t, db, "admin")
	memberID := createTestUser(t, db, "member")

	group, err := chatRepo.CreateGroup(ctx, "ops", adminID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	err = memberRepo.Add(ctx, &domain.ChatMember{
		ChatID: group.ID,
		UserID: memberID,
		Role:   domain.RoleMember,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := memberRepo.Get(ctx, group.ID, memberID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Role != domain.RoleMember {
		t.Fatalf("role = %q, want member", got.Role)
	}

	err = memberRepo.Remove(ctx, group.ID, memberID)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	_, err = memberRepo.Get(ctx, group.ID, memberID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get after remove error = %v, want ErrNotFound", err)
	}
}
