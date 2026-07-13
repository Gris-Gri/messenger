package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"messenger/internal/domain"
)

// HubReadNotifier adapts Hub to domain.RealtimeNotifier.
type HubReadNotifier struct {
	hub     *Hub
	members domain.MemberRepository
	logger  *slog.Logger
}

func NewHubReadNotifier(hub *Hub, members domain.MemberRepository, logger *slog.Logger) *HubReadNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &HubReadNotifier{hub: hub, members: members, logger: logger}
}

func (n *HubReadNotifier) NotifyRead(ctx context.Context, chatID, userID, lastReadMessageID int64) {
	memberIDs, err := n.members.ListUserIDs(ctx, chatID)
	if err != nil {
		n.logger.Warn("read broadcast skipped", "err", err, "chat_id", chatID)
		return
	}

	frame := readFrame{
		Type:              FrameTypeRead,
		ChatID:            chatID,
		UserID:            userID,
		LastReadMessageID: lastReadMessageID,
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("read broadcast marshal failed", "err", err, "chat_id", chatID)
		return
	}

	n.hub.BroadcastRead(chatID, userID, payload, memberIDs)
}

func (n *HubReadNotifier) NotifyChatUpdated(ctx context.Context, chatID, actorUserID int64, title string) {
	memberIDs, err := n.members.ListUserIDs(ctx, chatID)
	if err != nil {
		n.logger.Warn("chat_updated broadcast skipped", "err", err, "chat_id", chatID)
		return
	}

	frame := chatUpdatedFrame{
		Type:   FrameTypeChatUpdated,
		ChatID: chatID,
		Title:  title,
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("chat_updated broadcast marshal failed", "err", err, "chat_id", chatID)
		return
	}

	n.hub.BroadcastChatUpdated(chatID, actorUserID, payload, memberIDs)
}

func (n *HubReadNotifier) NotifyUserUpdated(ctx context.Context, userID int64, login string) {
	recipientIDs, err := n.members.ListSharedChatUserIDs(ctx, userID)
	if err != nil {
		n.logger.Warn("user_updated broadcast skipped", "err", err, "user_id", userID)
		return
	}

	frame := userUpdatedFrame{
		Type:   FrameTypeUserUpdated,
		UserID: userID,
		Login:  login,
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("user_updated broadcast marshal failed", "err", err, "user_id", userID)
		return
	}

	n.hub.BroadcastToUsers(payload, recipientIDs)
}

func (n *HubReadNotifier) NotifyPresence(ctx context.Context, userID int64, status string, lastSeenAt *time.Time) {
	recipientIDs, err := n.members.ListSharedChatUserIDs(ctx, userID)
	if err != nil {
		n.logger.Warn("presence broadcast skipped", "err", err, "user_id", userID)
		return
	}

	frame := presenceFrame{
		Type:   FrameTypePresence,
		UserID: userID,
		Status: status,
	}
	if lastSeenAt != nil {
		formatted := formatTime(*lastSeenAt)
		frame.LastSeenAt = &formatted
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("presence broadcast marshal failed", "err", err, "user_id", userID)
		return
	}

	n.hub.BroadcastToUsers(payload, recipientIDs)
}

func (n *HubReadNotifier) NotifyMessageEdited(ctx context.Context, chatID, messageID int64, body string, editedAt time.Time) {
	memberIDs, err := n.members.ListUserIDs(ctx, chatID)
	if err != nil {
		n.logger.Warn("message_edited broadcast skipped", "err", err, "chat_id", chatID)
		return
	}

	frame := messageEditedFrame{
		Type:      FrameTypeMessageEdited,
		ChatID:    chatID,
		MessageID: messageID,
		Body:      body,
		EditedAt:  formatTime(editedAt),
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("message_edited broadcast marshal failed", "err", err, "chat_id", chatID)
		return
	}

	n.hub.BroadcastToUsers(payload, memberIDs)
}

func (n *HubReadNotifier) NotifyReactionUpdated(ctx context.Context, chatID, messageID int64, counts domain.ReactionCounts) {
	memberIDs, err := n.members.ListUserIDs(ctx, chatID)
	if err != nil {
		n.logger.Warn("reaction_updated broadcast skipped", "err", err, "chat_id", chatID)
		return
	}

	frame := reactionUpdatedFrame{
		Type:      FrameTypeReactionUpdated,
		ChatID:    chatID,
		MessageID: messageID,
		Reactions: reactionCountsPayload{
			Like:    counts.Like,
			Dislike: counts.Dislike,
			Heart:   counts.Heart,
		},
	}
	payload, err := json.Marshal(frame)
	if err != nil {
		n.logger.Warn("reaction_updated broadcast marshal failed", "err", err, "chat_id", chatID)
		return
	}

	n.hub.BroadcastToUsers(payload, memberIDs)
}

var _ domain.RealtimeNotifier = (*HubReadNotifier)(nil)
