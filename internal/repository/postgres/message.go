package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"messenger/internal/domain"
)

type MessageRepository struct {
	db *DB
}

func (r *MessageRepository) Create(ctx context.Context, msg *domain.Message) (*domain.Message, error) {
	const q = `
		INSERT INTO messages (chat_id, sender_id, client_msg_id, body)
		VALUES ($1, $2, $3::uuid, $4)
		ON CONFLICT (chat_id, client_msg_id) DO UPDATE
		    SET body = messages.body
		RETURNING id, chat_id, sender_id, client_msg_id::text, body, created_at
	`

	var created domain.Message
	err := r.db.pool.QueryRow(ctx, q, msg.ChatID, msg.SenderID, msg.ClientMsgID, msg.Body).Scan(
		&created.ID,
		&created.ChatID,
		&created.SenderID,
		&created.ClientMsgID,
		&created.Body,
		&created.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &created, nil
}

func (r *MessageRepository) ListByChat(ctx context.Context, chatID, beforeID int64, limit int) ([]domain.Message, error) {
	if limit <= 0 {
		return []domain.Message{}, nil
	}

	var (
		rows pgx.Rows
		err  error
	)

	if beforeID <= 0 {
		const q = `
			SELECT id, chat_id, sender_id, client_msg_id::text, body, created_at
			FROM messages
			WHERE chat_id = $1
			ORDER BY id DESC
			LIMIT $2
		`
		rows, err = r.db.pool.Query(ctx, q, chatID, limit)
	} else {
		const q = `
			SELECT id, sender_id, body, created_at
			FROM messages
			WHERE chat_id = $1 AND id < $2
			ORDER BY id DESC
			LIMIT $3
		`
		rows, err = r.db.pool.Query(ctx, q, chatID, beforeID, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	messages, err := scanMessageRows(rows, chatID, beforeID <= 0)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (r *MessageRepository) Search(ctx context.Context, chatID int64, query string) ([]domain.Message, error) {
	const q = `
		SELECT id, sender_id, body, created_at
		FROM messages
		WHERE chat_id = $1 AND body ILIKE '%' || $2 || '%'
		ORDER BY id DESC
		LIMIT 50
	`

	rows, err := r.db.pool.Query(ctx, q, chatID, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	return scanMessageRows(rows, chatID, false)
}

func scanMessageRows(rows pgx.Rows, chatID int64, fullColumns bool) ([]domain.Message, error) {
	var messages []domain.Message
	for rows.Next() {
		var m domain.Message
		var err error
		if fullColumns {
			err = rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.ClientMsgID, &m.Body, &m.CreatedAt)
		} else {
			m.ChatID = chatID
			err = rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt)
		}
		if err != nil {
			return nil, fmt.Errorf("postgres: scan message: %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	if messages == nil {
		messages = []domain.Message{}
	}

	return messages, nil
}

var _ domain.MessageRepository = (*MessageRepository)(nil)
