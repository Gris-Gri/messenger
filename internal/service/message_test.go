package service

import (
	"context"
	"testing"
	"time"

	"messenger/internal/domain"
)

func TestService_SendMessageIdempotentByClientMsgID(t *testing.T) {
	t.Parallel()

	const (
		chatID   int64 = 5
		callerID int64 = 1
	)

	clientMsgID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	stored := &domain.Message{
		ID:          100,
		ChatID:      chatID,
		SenderID:    callerID,
		ClientMsgID: clientMsgID,
		Body:        "hello",
	}

	createCalls := 0
	messages := &mockMessageRepo{
		createFn: func(_ context.Context, msg *domain.Message) (*domain.Message, error) {
			createCalls++
			if msg.ClientMsgID != clientMsgID {
				t.Fatalf("client_msg_id = %q, want %q", msg.ClientMsgID, clientMsgID)
			}
			return stored, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())

	first, err := svc.SendMessage(context.Background(), callerID, chatID, clientMsgID, "hello")
	if err != nil {
		t.Fatalf("first SendMessage: %v", err)
	}

	second, err := svc.SendMessage(context.Background(), callerID, chatID, clientMsgID, "hello")
	if err != nil {
		t.Fatalf("second SendMessage: %v", err)
	}

	if first.ID != second.ID {
		t.Fatalf("idempotent ids differ: %d vs %d", first.ID, second.ID)
	}
	if createCalls != 2 {
		t.Fatalf("Create calls = %d, want 2 (service delegates idempotency to repository)", createCalls)
	}
}

func TestService_SendMessageNotMemberForbidden(t *testing.T) {
	t.Parallel()

	members := &mockMemberRepo{
		getFn: func(context.Context, int64, int64) (*domain.ChatMember, error) {
			return nil, domain.ErrNotFound
		},
	}
	messages := &mockMessageRepo{
		createFn: func(context.Context, *domain.Message) (*domain.Message, error) {
			t.Fatal("Create must not be called for non-member")
			return nil, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.SendMessage(context.Background(), 1, 2, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "hi")
	if err != domain.ErrForbidden {
		t.Fatalf("SendMessage error = %v, want ErrForbidden", err)
	}
}

func TestService_EditMessageHappyPath(t *testing.T) {
	t.Parallel()

	const (
		chatID    int64 = 5
		callerID  int64 = 1
		messageID int64 = 100
	)

	editedAt := time.Now().UTC()
	notifier := &mockRealtimeNotifier{}
	messages := &mockMessageRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Message, error) {
			return &domain.Message{
				ID:       id,
				ChatID:   chatID,
				SenderID: callerID,
				Body:     "old",
			}, nil
		},
		updateMessageBodyFn: func(_ context.Context, id, senderID int64, newBody string) (*domain.Message, error) {
			if id != messageID || senderID != callerID || newBody != "новый текст" {
				t.Fatalf("UpdateMessageBody(%d, %d, %q)", id, senderID, newBody)
			}
			return &domain.Message{
				ID:       id,
				ChatID:   chatID,
				SenderID: senderID,
				Body:     newBody,
				EditedAt: &editedAt,
			}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, notifier, testJWTManager())

	got, err := svc.EditMessage(context.Background(), callerID, chatID, messageID, "  новый текст  ")
	if err != nil {
		t.Fatalf("EditMessage: %v", err)
	}
	if got.Body != "новый текст" || got.EditedAt == nil {
		t.Fatalf("got = %+v", got)
	}
	if len(notifier.messageEditedCalls) != 1 {
		t.Fatalf("NotifyMessageEdited calls = %d, want 1", len(notifier.messageEditedCalls))
	}
	call := notifier.messageEditedCalls[0]
	if call.chatID != chatID || call.messageID != messageID || call.body != "новый текст" {
		t.Fatalf("notify call = %+v", call)
	}
}

func TestService_EditMessageForeignForbidden(t *testing.T) {
	t.Parallel()

	messages := &mockMessageRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Message, error) {
			return &domain.Message{ID: id, ChatID: 5, SenderID: 2, Body: "old"}, nil
		},
		updateMessageBodyFn: func(context.Context, int64, int64, string) (*domain.Message, error) {
			t.Fatal("UpdateMessageBody must not be called for foreign message")
			return nil, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleAdmin}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.EditMessage(context.Background(), 1, 5, 100, "hack")
	if err != domain.ErrForbidden {
		t.Fatalf("EditMessage error = %v, want ErrForbidden", err)
	}
}

func TestService_EditMessageEmptyBodyValidation(t *testing.T) {
	t.Parallel()

	messages := &mockMessageRepo{
		getByIDFn: func(context.Context, int64) (*domain.Message, error) {
			t.Fatal("GetByID must not be called for empty body")
			return nil, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(context.Context, int64, int64) (*domain.ChatMember, error) {
			t.Fatal("ensureChatMember must not be called for empty body")
			return nil, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())

	_, err := svc.EditMessage(context.Background(), 1, 5, 100, "   ")
	if err != domain.ErrValidation {
		t.Fatalf("EditMessage error = %v, want ErrValidation", err)
	}
}

func TestService_SetMessageReactionToggleAndNotify(t *testing.T) {
	t.Parallel()

	const (
		chatID    int64 = 5
		callerID  int64 = 1
		messageID int64 = 100
	)

	notifier := &mockRealtimeNotifier{}
	messages := &mockMessageRepo{
		getByIDFn: func(_ context.Context, id int64) (*domain.Message, error) {
			return &domain.Message{ID: id, ChatID: chatID, SenderID: 2, Body: "hi"}, nil
		},
		toggleReactionFn: func(_ context.Context, msgID, userID int64, reaction string) (domain.ReactionSummary, error) {
			if msgID != messageID || userID != callerID || reaction != domain.ReactionLike {
				t.Fatalf("ToggleReaction(%d, %d, %q)", msgID, userID, reaction)
			}
			like := domain.ReactionLike
			return domain.ReactionSummary{Like: 1, MyReaction: &like}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, notifier, testJWTManager())

	got, err := svc.SetMessageReaction(context.Background(), callerID, chatID, messageID, domain.ReactionLike)
	if err != nil {
		t.Fatalf("SetMessageReaction: %v", err)
	}
	if got.Like != 1 || got.MyReaction == nil || *got.MyReaction != domain.ReactionLike {
		t.Fatalf("summary = %+v", got)
	}
	if len(notifier.reactionUpdatedCalls) != 1 {
		t.Fatalf("NotifyReactionUpdated calls = %d, want 1", len(notifier.reactionUpdatedCalls))
	}
	call := notifier.reactionUpdatedCalls[0]
	if call.chatID != chatID || call.messageID != messageID || call.counts.Like != 1 {
		t.Fatalf("notify call = %+v", call)
	}
}

func TestService_GetMessageHistoryIncludesReactions(t *testing.T) {
	t.Parallel()

	like := domain.ReactionLike
	messages := &mockMessageRepo{
		listByChatFn: func(_ context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
			return []domain.Message{
				{ID: 10, ChatID: chatID, SenderID: 1, Body: "a"},
				{ID: 11, ChatID: chatID, SenderID: 2, Body: "b"},
			}, nil
		},
		getReactionSummariesFn: func(_ context.Context, messageIDs []int64, viewerID int64) (map[int64]domain.ReactionSummary, error) {
			if len(messageIDs) != 2 || viewerID != 1 {
				t.Fatalf("GetReactionSummaries(%v, %d)", messageIDs, viewerID)
			}
			return map[int64]domain.ReactionSummary{
				10: {Like: 2, Dislike: 1, Heart: 0, MyReaction: &like},
			}, nil
		},
	}
	members := &mockMemberRepo{
		getFn: func(_ context.Context, cID, userID int64) (*domain.ChatMember, error) {
			return &domain.ChatMember{ChatID: cID, UserID: userID, Role: domain.RoleMember}, nil
		},
	}

	svc := New(&mockUserRepo{}, &mockChatRepo{}, messages, members, &mockReadStateRepo{}, nil, testJWTManager())

	got, err := svc.GetMessageHistory(context.Background(), 1, 5, 0, 50)
	if err != nil {
		t.Fatalf("GetMessageHistory: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Reactions.Like != 2 || got[0].Reactions.MyReaction == nil || *got[0].Reactions.MyReaction != domain.ReactionLike {
		t.Fatalf("msg 10 reactions = %+v", got[0].Reactions)
	}
	if got[1].Reactions.Like != 0 || got[1].Reactions.MyReaction != nil {
		t.Fatalf("msg 11 reactions = %+v, want zeros", got[1].Reactions)
	}
}
