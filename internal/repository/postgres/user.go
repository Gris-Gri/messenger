package postgres

import (
	"context"

	"messenger/internal/domain"
)

type UserRepository struct {
	db *DB
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (*domain.User, error) {
	const q = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id, login, password_hash, created_at
	`

	var u domain.User
	err := r.db.pool.QueryRow(ctx, q, login, passwordHash).Scan(
		&u.ID,
		&u.Login,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &u, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	const q = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1
	`

	var u domain.User
	err := r.db.pool.QueryRow(ctx, q, login).Scan(
		&u.ID,
		&u.Login,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	const q = `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE id = $1
	`

	var u domain.User
	err := r.db.pool.QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.Login,
		&u.PasswordHash,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, mapError(err)
	}

	return &u, nil
}

func (r *UserRepository) SearchByLogin(ctx context.Context, query string, excludeUserID int64, limit int) ([]domain.User, error) {
	const q = `
		SELECT id, login
		FROM users
		WHERE login ILIKE '%' || $1 || '%'
		  AND id <> $2
		ORDER BY login ASC
		LIMIT $3
	`

	rows, err := r.db.pool.Query(ctx, q, query, excludeUserID, limit)
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Login); err != nil {
			return nil, mapError(err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, mapError(err)
	}

	return users, nil
}

var _ domain.UserRepository = (*UserRepository)(nil)
