package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"messenger/internal/domain"
	"messenger/pkg/password"
)

func TestRegisterHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.createFn = func(_ context.Context, login, _ string) (*domain.User, error) {
		return &domain.User{ID: 1, Login: login, CreatedAt: time.Now()}, nil
	}

	resp, data := env.do(http.MethodPost, "/register", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusCreated)

	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["login"] != "alice" {
		t.Fatalf("login = %v", body["login"])
	}
}

func TestRegisterValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/register", map[string]string{
		"login":    "",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestLoginHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	hash, err := passwordHash("secret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	env.users.getByLoginFn = func(_ context.Context, login string) (*domain.User, error) {
		return &domain.User{ID: 1, Login: login, PasswordHash: hash}, nil
	}

	resp, data := env.do(http.MethodPost, "/login", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Fatal("expected tokens")
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByLoginFn = func(_ context.Context, _ string) (*domain.User, error) {
		return nil, domain.ErrNotFound
	}

	resp, data := env.do(http.MethodPost, "/login", map[string]string{
		"login":    "alice",
		"password": "secret",
	}, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "invalid_credentials")
}

func TestRefreshHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id, Login: "alice"}, nil
	}

	pair, err := testJWTManager().IssuePair(1)
	if err != nil {
		t.Fatalf("IssuePair: %v", err)
	}

	resp, data := env.do(http.MethodPost, "/refresh", nil, pair.RefreshToken)
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestRefreshUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/refresh", nil, "bad-token")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestListChatsHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	now := time.Now()
	title := "team"
	env.chats.listByUserFn = func(_ context.Context, userID int64) ([]domain.ChatListItem, error) {
		if userID != 1 {
			t.Fatalf("userID = %d", userID)
		}
		return []domain.ChatListItem{{
			ID:    10,
			Type:  domain.ChatTypeGroup,
			Title: &title,
			LastMessageAt: func() *time.Time {
				v := now
				return &v
			}(),
		}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestListChatsUnauthorized(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodGet, "/chats", nil, "")
	assertStatus(t, resp, http.StatusUnauthorized)
	assertErrorCode(t, data, "unauthorized")
}

func TestCreateChatDirectHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id}, nil
	}
	env.chats.createDirectFn = func(_ context.Context, a, b int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 5, Type: domain.ChatTypeDirect, UserAID: &a, UserBID: &b, CreatedAt: time.Now()}, nil
	}

	resp, _ := env.do(http.MethodPost, "/chats", map[string]any{
		"type":    "direct",
		"user_id": 2,
	}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusCreated)
}

func TestCreateChatValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodPost, "/chats", map[string]string{
		"type": "group",
	}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func TestListMessagesHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleMember}, nil
	}
	env.messages.listByChatFn = func(_ context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
		return []domain.Message{{ID: 100, SenderID: 1, Body: "hi", CreatedAt: time.Now()}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats/1/messages?limit=10", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestListMessagesForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
		return nil, domain.ErrNotFound
	}

	resp, data := env.do(http.MethodGet, "/chats/1/messages", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusForbidden)
	assertErrorCode(t, data, "forbidden")
}

func TestAddMemberHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		if userID == 1 {
			return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
		}
		return nil, domain.ErrNotFound
	}
	env.users.getByIDFn = func(_ context.Context, id int64) (*domain.User, error) {
		return &domain.User{ID: id}, nil
	}
	env.members.addFn = func(_ context.Context, _ *domain.ChatMember) error { return nil }

	resp, _ := env.do(http.MethodPost, "/chats/1/members", map[string]int64{"user_id": 2}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNoContent)
}

func TestAddMemberForbidden(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, _, _ int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{Role: domain.RoleMember}, nil
	}

	resp, data := env.do(http.MethodPost, "/chats/1/members", map[string]int64{"user_id": 2}, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusForbidden)
	assertErrorCode(t, data, "forbidden")
}

func TestRemoveMemberHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
	}
	env.members.removeFn = func(_ context.Context, _, _ int64) error { return nil }

	resp, _ := env.do(http.MethodDelete, "/chats/1/members/2", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNoContent)
}

func TestRemoveMemberNotFound(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.chats.getByIDFn = func(_ context.Context, _ int64) (*domain.Chat, error) {
		return &domain.Chat{ID: 1, Type: domain.ChatTypeGroup}, nil
	}
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID, Role: domain.RoleAdmin}, nil
	}
	env.members.removeFn = func(_ context.Context, _, _ int64) error {
		return domain.ErrNotFound
	}

	resp, data := env.do(http.MethodDelete, "/chats/1/members/2", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusNotFound)
	assertErrorCode(t, data, "not_found")
}

func TestSearchHappyPath(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.members.getFn = func(_ context.Context, chatID, userID int64) (*domain.ChatMember, error) {
		return &domain.ChatMember{ChatID: chatID, UserID: userID}, nil
	}
	env.messages.searchFn = func(_ context.Context, _ int64, query string) ([]domain.Message, error) {
		if query != "hello" {
			t.Fatalf("query = %q", query)
		}
		return []domain.Message{{ID: 1, SenderID: 1, Body: "hello", CreatedAt: time.Now()}}, nil
	}

	resp, _ := env.do(http.MethodGet, "/chats/1/search?q=hello", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusOK)
}

func TestSearchValidationError(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	resp, data := env.do(http.MethodGet, "/chats/1/search", nil, env.accessToken(t, 1))
	assertStatus(t, resp, http.StatusBadRequest)
	assertErrorCode(t, data, "validation_error")
}

func passwordHash(raw string) (string, error) {
	return password.Hash(raw)
}
