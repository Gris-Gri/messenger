package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"messenger/internal/domain"
)

func TestMessageRepository_CreateListSearch(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	msgRepo := NewMessageRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice")
	bobID := createTestUser(t, db, "bob")

	chat, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	firstID := uuid.NewString()
	secondID := uuid.NewString()

	first, err := msgRepo.Create(ctx, &domain.Message{
		ChatID:      chat.ID,
		SenderID:    aliceID,
		ClientMsgID: firstID,
		Body:        "first message",
	})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}

	second, err := msgRepo.Create(ctx, &domain.Message{
		ChatID:      chat.ID,
		SenderID:    bobID,
		ClientMsgID: secondID,
		Body:        "second message with keyword",
	})
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	page, err := msgRepo.ListByChat(ctx, chat.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListByChat first page: %v", err)
	}
	if len(page) != 2 || page[0].ID != second.ID {
		t.Fatalf("first page = %+v, want newest first", page)
	}

	older, err := msgRepo.ListByChat(ctx, chat.ID, second.ID, 10)
	if err != nil {
		t.Fatalf("ListByChat cursor page: %v", err)
	}
	if len(older) != 1 || older[0].ID != first.ID {
		t.Fatalf("cursor page = %+v, want only first message", older)
	}

	found, err := msgRepo.Search(ctx, chat.ID, "keyword")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(found) != 1 || found[0].ID != second.ID {
		t.Fatalf("search result = %+v, want second message", found)
	}
}

func TestMessageRepository_CreateIdempotentByClientMsgID(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	msgRepo := NewMessageRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice")
	bobID := createTestUser(t, db, "bob")

	chat, err := chatRepo.CreateDirect(ctx, aliceID, bobID)
	if err != nil {
		t.Fatalf("CreateDirect: %v", err)
	}

	clientMsgID := uuid.NewString()
	original := &domain.Message{
		ChatID:      chat.ID,
		SenderID:    aliceID,
		ClientMsgID: clientMsgID,
		Body:        "hello",
	}

	first, err := msgRepo.Create(ctx, original)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	retry := &domain.Message{
		ChatID:      chat.ID,
		SenderID:    aliceID,
		ClientMsgID: clientMsgID,
		Body:        "hello",
	}
	second, err := msgRepo.Create(ctx, retry)
	if err != nil {
		t.Fatalf("second Create: %v", err)
	}

	if second.ID != first.ID {
		t.Fatalf("idempotent Create id = %d, want %d", second.ID, first.ID)
	}
	if second.ClientMsgID != clientMsgID {
		t.Fatalf("ClientMsgID = %q, want %q", second.ClientMsgID, clientMsgID)
	}

	all, err := msgRepo.ListByChat(ctx, chat.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListByChat: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(all))
	}
}
