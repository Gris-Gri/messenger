package domain

import (
	"context"
	"time"
)

// RealtimeNotifier pushes realtime events to online clients.
// Implementations live in the transport layer (WS hub); service must not import handlers.
type RealtimeNotifier interface {
	NotifyRead(ctx context.Context, chatID, userID, lastReadMessageID int64)
	NotifyChatUpdated(ctx context.Context, chatID, actorUserID int64, title string)
	NotifyUserUpdated(ctx context.Context, userID int64, login string)
	NotifyPresence(ctx context.Context, userID int64, status string, lastSeenAt *time.Time)
}

// PresenceChecker reports whether a user currently has an active WS connection.
// Implemented by the WS hub; service must not import handlers.
type PresenceChecker interface {
	IsOnline(userID int64) bool
}
