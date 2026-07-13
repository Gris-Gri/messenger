package ws

import "time"

const (
	FrameTypeAck             = "ack"
	FrameTypeNewMessage      = "new_message"
	FrameTypeSendMessage     = "send_message"
	FrameTypeRead            = "read"
	FrameTypeChatUpdated     = "chat_updated"
	FrameTypeUserUpdated     = "user_updated"
	FrameTypePresence        = "presence"
	FrameTypeMessageEdited   = "message_edited"
	FrameTypeReactionUpdated = "reaction_updated"
)

type authFrame struct {
	Token string `json:"token"`
}

type sendMessageFrame struct {
	Type        string `json:"type"`
	ChatID      int64  `json:"chat_id"`
	ClientMsgID string `json:"client_msg_id"`
	Body        string `json:"body"`
}

type ackFrame struct {
	Type        string `json:"type"`
	ClientMsgID string `json:"client_msg_id"`
	ServerID    int64  `json:"server_id"`
}

type newMessageFrame struct {
	Type    string         `json:"type"`
	ChatID  int64          `json:"chat_id"`
	Message messagePayload `json:"message"`
}

type readFrame struct {
	Type              string `json:"type"`
	ChatID            int64  `json:"chat_id"`
	UserID            int64  `json:"user_id"`
	LastReadMessageID int64  `json:"last_read_message_id"`
}

type chatUpdatedFrame struct {
	Type   string `json:"type"`
	ChatID int64  `json:"chat_id"`
	Title  string `json:"title"`
}

type userUpdatedFrame struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
}

type presenceFrame struct {
	Type       string  `json:"type"`
	UserID     int64   `json:"user_id"`
	Status     string  `json:"status"`
	LastSeenAt *string `json:"last_seen_at,omitempty"`
}

type messageEditedFrame struct {
	Type      string `json:"type"`
	ChatID    int64  `json:"chat_id"`
	MessageID int64  `json:"message_id"`
	Body      string `json:"body"`
	EditedAt  string `json:"edited_at"`
}

type reactionCountsPayload struct {
	Like    int `json:"like"`
	Dislike int `json:"dislike"`
	Heart   int `json:"heart"`
}

type reactionUpdatedFrame struct {
	Type      string                `json:"type"`
	ChatID    int64                 `json:"chat_id"`
	MessageID int64                 `json:"message_id"`
	Reactions reactionCountsPayload `json:"reactions"`
}

type messagePayload struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

const timeRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func formatTime(t time.Time) string {
	return t.UTC().Format(timeRFC3339Nano)
}
