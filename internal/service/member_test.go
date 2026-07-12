package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"messenger/internal/domain"
)

func TestService_AddMemberNonAdminForbidden(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 10
		adminID  int64 = 1
		memberID int64 = 2
		newUser  int64 = 3
	)

	chats := &mockChatRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Chat, error) {
			if id != chatID {
				t.Fatalf("chat id = %d, want %d", id, chatID)
			}
			return &domain.Chat{ID: chatID, Type: domain.ChatTypeGroup}, nil
		},
	}

	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			if cID != chatID {
				t.Fatalf("chat id = %d, want %d", cID, chatID)
			}
			switch userID {
			case adminID:
				return &domain.ChatMember{ChatID: chatID, UserID: adminID, Role: domain.RoleMember}, nil
			case memberID:
				return &domain.ChatMember{ChatID: chatID, UserID: memberID, Role: domain.RoleAdmin}, nil
			default:
				return nil, domain.ErrNotFound
			}
		},
		addFn: func(context.Context, *domain.ChatMember) error {
			t.Fatal("Add must not be called for non-admin caller")
			return nil
		},
	}

	svc := New(&mockUserRepo{}, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	err := svc.AddMember(context.Background(), adminID, chatID, newUser)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("AddMember error = %v, want ErrForbidden", err)
	}
}

func TestService_AddMemberAdminSuccess(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 10
		adminID  int64 = 1
		newUser  int64 = 3
	)

	var added bool
	chats := &mockChatRepo{
		getByIDFn: func(context.Context, int64) (*domain.Chat, error) {
			return &domain.Chat{ID: chatID, Type: domain.ChatTypeGroup}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, _, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
		},
		addFn: func(_ context.Context, member *domain.ChatMember) error {
			if member.UserID != newUser || member.Role != domain.RoleMember {
				t.Fatalf("unexpected member: %+v", member)
			}
			added = true
			return nil
		},
	}
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			if id != newUser {
				t.Fatalf("user id = %d, want %d", id, newUser)
			}
			return &domain.User{ID: newUser}, nil
		},
	}

	svc := New(users, chats, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager())

	if err := svc.AddMember(context.Background(), adminID, chatID, newUser); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if !added {
		t.Fatal("expected member to be added")
	}
}

type stubPresence map[int64]bool

func (s stubPresence) IsOnline(userID int64) bool {
	return s[userID]
}

func TestService_ListMembersEnrichesOnline(t *testing.T) {
	t.Parallel()

	lastSeen := time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)
	members := &mockMemberRepo{
		getFn: func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
		},
		listByChatFn: func(_ context.Context, chatID int64) ([]domain.ChatMember, error) {
			return []domain.ChatMember{
				{ChatID: chatID, UserID: 1, Login: "alice", Role: domain.RoleAdmin},
				{ChatID: chatID, UserID: 2, Login: "bob", Role: domain.RoleMember, LastSeenAt: &lastSeen},
			}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, &mockMessageRepo{}, members, &mockReadStateRepo{}, nil, testJWTManager()).
		WithPresence(stubPresence{1: true})

	got, err := svc.ListMembers(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if !got[0].Online {
		t.Fatal("alice should be online")
	}
	if got[1].Online {
		t.Fatal("bob should be offline")
	}
	if got[1].LastSeenAt == nil || !got[1].LastSeenAt.Equal(lastSeen) {
		t.Fatalf("bob last_seen_at = %v", got[1].LastSeenAt)
	}
}
