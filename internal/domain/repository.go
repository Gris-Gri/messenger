package domain

import (
	"context"
	"time"
)

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*User, error)
	GetByLogin(ctx context.Context, login string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	SearchByLogin(ctx context.Context, query string, excludeUserID int64, limit int) ([]User, error)
	// UpdateLogin returns the user without password_hash.
	UpdateLogin(ctx context.Context, userID int64, login string) (*User, error)
	// UpdatePasswordHash does not return the hash.
	UpdatePasswordHash(ctx context.Context, userID int64, passwordHash string) error
	// UpdateLastSeenAt sets last_seen_at and returns the stored timestamp.
	UpdateLastSeenAt(ctx context.Context, userID int64, at time.Time) (time.Time, error)
}

type ChatRepository interface {
	CreateDirect(ctx context.Context, userAID, userBID int64) (*Chat, error)
	CreateGroup(ctx context.Context, title string, createdBy int64) (*Chat, error)
	UpdateChatTitle(ctx context.Context, chatID int64, title string) (*Chat, error)
	GetByID(ctx context.Context, id int64) (*Chat, error)
	GetDirectByUsers(ctx context.Context, userAID, userBID int64) (*Chat, error)
	ListByUser(ctx context.Context, userID int64) ([]ChatListItem, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) (*Message, error)
	GetByID(ctx context.Context, messageID int64) (*Message, error)
	UpdateMessageBody(ctx context.Context, messageID, senderID int64, newBody string) (*Message, error)
	ListByChat(ctx context.Context, chatID, beforeID int64, limit int) ([]Message, error)
	Search(ctx context.Context, chatID int64, query string) ([]Message, error)
	// ToggleReaction inserts, replaces, or removes the user's reaction (same reaction = toggle off).
	// Returns the updated summary for that viewer.
	ToggleReaction(ctx context.Context, messageID, userID int64, reaction string) (ReactionSummary, error)
	// GetReactionSummaries returns aggregates for the given message IDs in one GROUP BY query.
	// Missing IDs are omitted from the map (caller treats them as zero counts / no my_reaction).
	GetReactionSummaries(ctx context.Context, messageIDs []int64, viewerID int64) (map[int64]ReactionSummary, error)
}

type MemberRepository interface {
	Add(ctx context.Context, member *ChatMember) error
	Remove(ctx context.Context, chatID, userID int64) error
	Get(ctx context.Context, chatID, userID int64) (*ChatMember, error)
	ListUserIDs(ctx context.Context, chatID int64) ([]int64, error)
	ListByChat(ctx context.Context, chatID int64) ([]ChatMember, error)
	// ListSharedChatUserIDs returns distinct user IDs that share at least one chat with userID.
	ListSharedChatUserIDs(ctx context.Context, userID int64) ([]int64, error)
}

type ReadStateRepository interface {
	UpsertReadState(ctx context.Context, chatID, userID, messageID int64) (int64, error)
	GetReadState(ctx context.Context, chatID int64) ([]ChatReadState, error)
	IsReadByAll(ctx context.Context, chatID, messageID, excludeUserID int64) (bool, error)
}
