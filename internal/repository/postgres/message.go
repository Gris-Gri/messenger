package postgres

import (
	"context"
	"errors"
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
		RETURNING id, chat_id, sender_id, client_msg_id::text, body, created_at, edited_at
	`

	var created domain.Message
	err := r.db.pool.QueryRow(ctx, q, msg.ChatID, msg.SenderID, msg.ClientMsgID, msg.Body).Scan(
		&created.ID,
		&created.ChatID,
		&created.SenderID,
		&created.ClientMsgID,
		&created.Body,
		&created.CreatedAt,
		&created.EditedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &created, nil
}

func (r *MessageRepository) GetByID(ctx context.Context, messageID int64) (*domain.Message, error) {
	const q = `
		SELECT id, chat_id, sender_id, client_msg_id::text, body, created_at, edited_at
		FROM messages
		WHERE id = $1
	`

	var msg domain.Message
	err := r.db.pool.QueryRow(ctx, q, messageID).Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderID,
		&msg.ClientMsgID,
		&msg.Body,
		&msg.CreatedAt,
		&msg.EditedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &msg, nil
}

func (r *MessageRepository) UpdateMessageBody(ctx context.Context, messageID, senderID int64, newBody string) (*domain.Message, error) {
	const q = `
		UPDATE messages
		SET body = $3, edited_at = now()
		WHERE id = $1 AND sender_id = $2
		RETURNING id, chat_id, sender_id, client_msg_id::text, body, created_at, edited_at
	`

	var msg domain.Message
	err := r.db.pool.QueryRow(ctx, q, messageID, senderID, newBody).Scan(
		&msg.ID,
		&msg.ChatID,
		&msg.SenderID,
		&msg.ClientMsgID,
		&msg.Body,
		&msg.CreatedAt,
		&msg.EditedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &msg, nil
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
			SELECT id, chat_id, sender_id, client_msg_id::text, body, created_at, edited_at
			FROM messages
			WHERE chat_id = $1
			ORDER BY id DESC
			LIMIT $2
		`
		rows, err = r.db.pool.Query(ctx, q, chatID, limit)
	} else {
		const q = `
			SELECT id, sender_id, body, created_at, edited_at
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
		SELECT id, sender_id, body, created_at, edited_at
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
			err = rows.Scan(&m.ID, &m.ChatID, &m.SenderID, &m.ClientMsgID, &m.Body, &m.CreatedAt, &m.EditedAt)
		} else {
			m.ChatID = chatID
			err = rows.Scan(&m.ID, &m.SenderID, &m.Body, &m.CreatedAt, &m.EditedAt)
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

func (r *MessageRepository) ToggleReaction(ctx context.Context, messageID, userID int64, reaction string) (domain.ReactionSummary, error) {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return domain.ReactionSummary{}, fmt.Errorf("postgres: begin toggle reaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var current string
	err = tx.QueryRow(ctx, `
		SELECT reaction
		FROM message_reactions
		WHERE message_id = $1 AND user_id = $2
		FOR UPDATE
	`, messageID, userID).Scan(&current)

	switch {
	case err == nil && current == reaction:
		if _, err := tx.Exec(ctx, `
			DELETE FROM message_reactions
			WHERE message_id = $1 AND user_id = $2
		`, messageID, userID); err != nil {
			return domain.ReactionSummary{}, mapError(err)
		}
	case err == nil:
		if _, err := tx.Exec(ctx, `
			INSERT INTO message_reactions (message_id, user_id, reaction)
			VALUES ($1, $2, $3)
			ON CONFLICT (message_id, user_id) DO UPDATE
			    SET reaction = EXCLUDED.reaction, created_at = now()
		`, messageID, userID, reaction); err != nil {
			return domain.ReactionSummary{}, mapError(err)
		}
	case errors.Is(err, pgx.ErrNoRows):
		if _, err := tx.Exec(ctx, `
			INSERT INTO message_reactions (message_id, user_id, reaction)
			VALUES ($1, $2, $3)
			ON CONFLICT (message_id, user_id) DO UPDATE
			    SET reaction = EXCLUDED.reaction, created_at = now()
		`, messageID, userID, reaction); err != nil {
			return domain.ReactionSummary{}, mapError(err)
		}
	default:
		return domain.ReactionSummary{}, mapError(err)
	}

	summary, err := scanReactionSummary(ctx, tx, messageID, userID)
	if err != nil {
		return domain.ReactionSummary{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ReactionSummary{}, fmt.Errorf("postgres: commit toggle reaction: %w", err)
	}

	return summary, nil
}

func (r *MessageRepository) GetReactionSummaries(ctx context.Context, messageIDs []int64, viewerID int64) (map[int64]domain.ReactionSummary, error) {
	result := make(map[int64]domain.ReactionSummary)
	if len(messageIDs) == 0 {
		return result, nil
	}

	const q = `
		SELECT
			message_id,
			COUNT(*) FILTER (WHERE reaction = 'like')::int,
			COUNT(*) FILTER (WHERE reaction = 'dislike')::int,
			COUNT(*) FILTER (WHERE reaction = 'heart')::int,
			MAX(reaction) FILTER (WHERE user_id = $2)
		FROM message_reactions
		WHERE message_id = ANY($1)
		GROUP BY message_id
	`

	rows, err := r.db.pool.Query(ctx, q, messageIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			messageID  int64
			summary    domain.ReactionSummary
			myReaction *string
		)
		if err := rows.Scan(&messageID, &summary.Like, &summary.Dislike, &summary.Heart, &myReaction); err != nil {
			return nil, fmt.Errorf("postgres: scan reaction summary: %w", err)
		}
		summary.MyReaction = myReaction
		result[messageID] = summary
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}

	return result, nil
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func scanReactionSummary(ctx context.Context, q querier, messageID, viewerID int64) (domain.ReactionSummary, error) {
	const sql = `
		SELECT
			COUNT(*) FILTER (WHERE reaction = 'like')::int,
			COUNT(*) FILTER (WHERE reaction = 'dislike')::int,
			COUNT(*) FILTER (WHERE reaction = 'heart')::int,
			MAX(reaction) FILTER (WHERE user_id = $2)
		FROM message_reactions
		WHERE message_id = $1
	`

	var (
		summary    domain.ReactionSummary
		myReaction *string
	)
	err := q.QueryRow(ctx, sql, messageID, viewerID).Scan(
		&summary.Like,
		&summary.Dislike,
		&summary.Heart,
		&myReaction,
	)
	if err != nil {
		return domain.ReactionSummary{}, mapError(err)
	}
	summary.MyReaction = myReaction
	return summary, nil
}

var _ domain.MessageRepository = (*MessageRepository)(nil)
