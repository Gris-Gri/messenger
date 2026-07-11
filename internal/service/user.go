package service

import (
	"context"
	"strings"

	"messenger/internal/domain"
)

const (
	defaultUserSearchLimit = 20
	maxUserSearchLimit     = 50
	minUserSearchQueryLen  = 2
)

func (s *Service) SearchUsers(ctx context.Context, callerID int64, login string, limit int) ([]domain.User, error) {
	login = strings.TrimSpace(login)
	if len([]rune(login)) < minUserSearchQueryLen {
		return nil, domain.ErrValidation
	}

	if limit <= 0 {
		limit = defaultUserSearchLimit
	}
	if limit > maxUserSearchLimit {
		limit = maxUserSearchLimit
	}

	return s.users.SearchByLogin(ctx, login, callerID, limit)
}
