package service

import (
	"context"
	"errors"
	"testing"

	"messenger/internal/domain"
	"messenger/pkg/password"
)

func TestService_UpdateLoginConflict(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		updateLoginFn: func(_ context.Context, userID int64, login string) (*domain.User, error) {
			if userID != 1 || login != "taken" {
				t.Fatalf("unexpected args: %d %q", userID, login)
			}
			return nil, domain.ErrConflict
		},
	}
	notifier := &mockRealtimeNotifier{}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, notifier, testJWTManager())

	_, err := svc.UpdateLogin(context.Background(), 1, "taken")
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("error = %v, want ErrConflict", err)
	}
	if len(notifier.userUpdatedCalls) != 0 {
		t.Fatal("must not notify on conflict")
	}
}

func TestService_UpdateLoginNotifies(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		updateLoginFn: func(_ context.Context, userID int64, login string) (*domain.User, error) {
			return &domain.User{ID: userID, Login: login}, nil
		},
	}
	notifier := &mockRealtimeNotifier{}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, notifier, testJWTManager())

	user, err := svc.UpdateLogin(context.Background(), 1, "alice2")
	if err != nil {
		t.Fatalf("UpdateLogin: %v", err)
	}
	if user.Login != "alice2" {
		t.Fatalf("login = %q", user.Login)
	}
	if len(notifier.userUpdatedCalls) != 1 {
		t.Fatalf("userUpdatedCalls = %d, want 1", len(notifier.userUpdatedCalls))
	}
	call := notifier.userUpdatedCalls[0]
	if call.userID != 1 || call.login != "alice2" {
		t.Fatalf("unexpected notify call: %+v", call)
	}
}

func TestService_UpdatePasswordWrongCurrent(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("correct")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: hash}, nil
		},
		updatePasswordHashFn: func(context.Context, int64, string) error {
			t.Fatal("UpdatePasswordHash must not be called")
			return nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	err = svc.UpdatePassword(context.Background(), 1, "wrong", "newsecret")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestService_UpdatePasswordSuccess(t *testing.T) {
	t.Parallel()

	hash, err := password.Hash("oldsecret")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	var stored string
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: hash}, nil
		},
		updatePasswordHashFn: func(_ context.Context, userID int64, passwordHash string) error {
			if userID != 1 {
				t.Fatalf("userID = %d", userID)
			}
			stored = passwordHash
			return nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	if err := svc.UpdatePassword(context.Background(), 1, "oldsecret", "newsecret"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}
	ok, err := password.Verify("newsecret", stored)
	if err != nil || !ok {
		t.Fatalf("stored hash does not match new password: ok=%v err=%v", ok, err)
	}
}

func TestService_GetMeStripsPasswordHash(t *testing.T) {
	t.Parallel()

	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.User, error) {
			return &domain.User{ID: id, Login: "alice", PasswordHash: "secret-hash"}, nil
		},
	}
	svc := New(users, &mockChatRepo{}, &mockMessageRepo{}, &mockMemberRepo{}, &mockReadStateRepo{}, nil, testJWTManager())

	user, err := svc.GetMe(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if user.PasswordHash != "" {
		t.Fatal("PasswordHash must be cleared")
	}
}
