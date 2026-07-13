package domain

import "time"

type Message struct {
	ID          int64
	ChatID      int64
	SenderID    int64
	ClientMsgID string
	Body        string
	CreatedAt   time.Time
	EditedAt    *time.Time
}
