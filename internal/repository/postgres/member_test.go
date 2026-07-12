package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestMemberRepository_ListSharedChatUserIDs(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	memberRepo := NewMemberRepository(db)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice")
	bobID := createTestUser(t, db, "bob")
	carolID := createTestUser(t, db, "carol")
	_ = createTestUser(t, db, "dave")

	direct, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}
	_ = direct

	group, err := chatRepo.CreateGroup(ctx, "ops", aliceID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := memberRepo.Add(ctx, &domain.ChatMember{ChatID: group.ID, UserID: carolID, Role: domain.RoleMember}); err != nil {
		t.Fatalf("Add carol: %v", err)
	}

	at := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	if _, err := userRepo.UpdateLastSeenAt(ctx, bobID, at); err != nil {
		t.Fatalf("UpdateLastSeenAt: %v", err)
	}

	ids, err := memberRepo.ListSharedChatUserIDs(ctx, aliceID)
	if err != nil {
		t.Fatalf("ListSharedChatUserIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("ids = %v, want bob and carol", ids)
	}

	listed, err := memberRepo.ListByChat(ctx, direct.ID)
	if err != nil {
		t.Fatalf("ListByChat: %v", err)
	}
	foundBob := false
	for _, m := range listed {
		if m.UserID == bobID {
			foundBob = true
			if m.LastSeenAt == nil || !m.LastSeenAt.Equal(at) {
				t.Fatalf("bob last_seen_at = %v, want %v", m.LastSeenAt, at)
			}
		}
		if m.UserID == aliceID && m.LastSeenAt != nil {
			t.Fatalf("alice last_seen_at should be null, got %v", m.LastSeenAt)
		}
	}
	if !foundBob {
		t.Fatal("bob missing from ListByChat")
	}
}
