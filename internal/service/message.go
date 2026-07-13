package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"messenger/internal/domain"
)

func (s *Service) SendMessage(ctx context.Context, callerID, chatID int64, clientMsgID, body string) (*domain.Message, error) {
	clientMsgID = strings.TrimSpace(clientMsgID)
	body = strings.TrimSpace(body)
	if clientMsgID == "" || body == "" {
		return nil, domain.ErrValidation
	}
	if _, err := uuid.Parse(clientMsgID); err != nil {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	return s.messages.Create(ctx, &domain.Message{
		ChatID:      chatID,
		SenderID:    callerID,
		ClientMsgID: clientMsgID,
		Body:        body,
	})
}

func (s *Service) EditMessage(ctx context.Context, callerID, chatID, messageID int64, body string) (*domain.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	existing, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if existing.ChatID != chatID {
		return nil, domain.ErrNotFound
	}
	if existing.SenderID != callerID {
		return nil, domain.ErrForbidden
	}

	updated, err := s.messages.UpdateMessageBody(ctx, messageID, callerID, body)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil && updated.EditedAt != nil {
		s.notifier.NotifyMessageEdited(ctx, updated.ChatID, updated.ID, updated.Body, *updated.EditedAt)
	}

	return updated, nil
}

func (s *Service) GetMessageHistory(ctx context.Context, callerID, chatID, beforeID int64, limit int) ([]domain.MessageWithReactions, error) {
	if limit <= 0 {
		return nil, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return nil, err
	}

	messages, err := s.messages.ListByChat(ctx, chatID, beforeID, limit)
	if err != nil {
		return nil, err
	}

	ids := make([]int64, len(messages))
	for i, msg := range messages {
		ids[i] = msg.ID
	}

	summaries, err := s.messages.GetReactionSummaries(ctx, ids, callerID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.MessageWithReactions, 0, len(messages))
	for _, msg := range messages {
		item := domain.MessageWithReactions{Message: msg}
		if summary, ok := summaries[msg.ID]; ok {
			item.Reactions = summary
		}
		result = append(result, item)
	}

	return result, nil
}

func (s *Service) SetMessageReaction(ctx context.Context, callerID, chatID, messageID int64, reaction string) (domain.ReactionSummary, error) {
	if !domain.ValidReaction(reaction) {
		return domain.ReactionSummary{}, domain.ErrValidation
	}

	if err := s.ensureChatMember(ctx, chatID, callerID); err != nil {
		return domain.ReactionSummary{}, err
	}

	existing, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		return domain.ReactionSummary{}, err
	}
	if existing.ChatID != chatID {
		return domain.ReactionSummary{}, domain.ErrNotFound
	}

	summary, err := s.messages.ToggleReaction(ctx, messageID, callerID, reaction)
	if err != nil {
		return domain.ReactionSummary{}, err
	}

	if s.notifier != nil {
		s.notifier.NotifyReactionUpdated(ctx, chatID, messageID, domain.ReactionCounts{
			Like:    summary.Like,
			Dislike: summary.Dislike,
			Heart:   summary.Heart,
		})
	}

	return summary, nil
}
