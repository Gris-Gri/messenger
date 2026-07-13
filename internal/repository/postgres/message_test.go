package postgres

import (
	"context"
	"errors"
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

func TestMessageRepository_UpdateMessageBody(t *testing.T) {
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

	created, err := msgRepo.Create(ctx, &domain.Message{
		ChatID:      chat.ID,
		SenderID:    aliceID,
		ClientMsgID: uuid.NewString(),
		Body:        "original keyword-old",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.EditedAt != nil {
		t.Fatalf("EditedAt before edit = %v, want nil", created.EditedAt)
	}

	updated, err := msgRepo.UpdateMessageBody(ctx, created.ID, aliceID, "edited keyword-new")
	if err != nil {
		t.Fatalf("UpdateMessageBody: %v", err)
	}
	if updated.Body != "edited keyword-new" || updated.EditedAt == nil {
		t.Fatalf("updated = %+v", updated)
	}

	_, err = msgRepo.UpdateMessageBody(ctx, created.ID, bobID, "hack")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("foreign UpdateMessageBody error = %v, want ErrNotFound", err)
	}

	found, err := msgRepo.Search(ctx, chat.ID, "keyword-new")
	if err != nil {
		t.Fatalf("Search new: %v", err)
	}
	if len(found) != 1 || found[0].Body != "edited keyword-new" {
		t.Fatalf("search after edit = %+v", found)
	}

	oldHits, err := msgRepo.Search(ctx, chat.ID, "keyword-old")
	if err != nil {
		t.Fatalf("Search old: %v", err)
	}
	if len(oldHits) != 0 {
		t.Fatalf("old text still searchable: %+v", oldHits)
	}

	page, err := msgRepo.ListByChat(ctx, chat.ID, 0, 10)
	if err != nil {
		t.Fatalf("ListByChat: %v", err)
	}
	if len(page) != 1 || page[0].EditedAt == nil || page[0].Body != "edited keyword-new" {
		t.Fatalf("ListByChat after edit = %+v", page)
	}
}

func TestMessageRepository_ToggleReactionAndSummaries(t *testing.T) {
	db := newTestDB(t)
	chatRepo := NewChatRepository(db)
	msgRepo := NewMessageRepository(db)
	memberRepo := NewMemberRepository(db)
	ctx := context.Background()

	aliceID := createTestUser(t, db, "alice")
	bobID := createTestUser(t, db, "bob")
	carolID := createTestUser(t, db, "carol")

	group, err := chatRepo.CreateGroup(ctx, "reactions", aliceID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := memberRepo.Add(ctx, &domain.ChatMember{ChatID: group.ID, UserID: bobID, Role: domain.RoleMember}); err != nil {
		t.Fatalf("add bob: %v", err)
	}
	if err := memberRepo.Add(ctx, &domain.ChatMember{ChatID: group.ID, UserID: carolID, Role: domain.RoleMember}); err != nil {
		t.Fatalf("add carol: %v", err)
	}

	msg, err := msgRepo.Create(ctx, &domain.Message{
		ChatID:      group.ID,
		SenderID:    aliceID,
		ClientMsgID: uuid.NewString(),
		Body:        "react to me",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	summary, err := msgRepo.ToggleReaction(ctx, msg.ID, aliceID, domain.ReactionLike)
	if err != nil {
		t.Fatalf("alice like: %v", err)
	}
	if summary.Like != 1 || summary.MyReaction == nil || *summary.MyReaction != domain.ReactionLike {
		t.Fatalf("after alice like = %+v", summary)
	}

	// Same reaction again -> toggle off
	summary, err = msgRepo.ToggleReaction(ctx, msg.ID, aliceID, domain.ReactionLike)
	if err != nil {
		t.Fatalf("alice like toggle off: %v", err)
	}
	if summary.Like != 0 || summary.MyReaction != nil {
		t.Fatalf("after toggle off = %+v", summary)
	}

	summary, err = msgRepo.ToggleReaction(ctx, msg.ID, aliceID, domain.ReactionLike)
	if err != nil {
		t.Fatalf("alice like again: %v", err)
	}

	// Replace like with heart — still one row for alice
	summary, err = msgRepo.ToggleReaction(ctx, msg.ID, aliceID, domain.ReactionHeart)
	if err != nil {
		t.Fatalf("alice heart replace: %v", err)
	}
	if summary.Like != 0 || summary.Heart != 1 || summary.MyReaction == nil || *summary.MyReaction != domain.ReactionHeart {
		t.Fatalf("after replace = %+v", summary)
	}

	var aliceRows int
	if err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM message_reactions WHERE message_id = $1 AND user_id = $2
	`, msg.ID, aliceID).Scan(&aliceRows); err != nil {
		t.Fatalf("count alice rows: %v", err)
	}
	if aliceRows != 1 {
		t.Fatalf("alice reaction rows = %d, want 1", aliceRows)
	}

	if _, err := msgRepo.ToggleReaction(ctx, msg.ID, bobID, domain.ReactionLike); err != nil {
		t.Fatalf("bob like: %v", err)
	}
	if _, err := msgRepo.ToggleReaction(ctx, msg.ID, carolID, domain.ReactionDislike); err != nil {
		t.Fatalf("carol dislike: %v", err)
	}

	summaries, err := msgRepo.GetReactionSummaries(ctx, []int64{msg.ID}, bobID)
	if err != nil {
		t.Fatalf("GetReactionSummaries: %v", err)
	}
	got, ok := summaries[msg.ID]
	if !ok {
		t.Fatal("missing summary for message")
	}
	if got.Like != 1 || got.Dislike != 1 || got.Heart != 1 {
		t.Fatalf("aggregates = %+v, want like=1 dislike=1 heart=1", got)
	}
	if got.MyReaction == nil || *got.MyReaction != domain.ReactionLike {
		t.Fatalf("bob my_reaction = %v, want like", got.MyReaction)
	}

	aliceView, err := msgRepo.GetReactionSummaries(ctx, []int64{msg.ID}, aliceID)
	if err != nil {
		t.Fatalf("alice summaries: %v", err)
	}
	if aliceView[msg.ID].MyReaction == nil || *aliceView[msg.ID].MyReaction != domain.ReactionHeart {
		t.Fatalf("alice my_reaction = %v, want heart", aliceView[msg.ID].MyReaction)
	}
}
