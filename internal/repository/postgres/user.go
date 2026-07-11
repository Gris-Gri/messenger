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

var _ domain.UserRepository = (*UserRepository)(nil)
