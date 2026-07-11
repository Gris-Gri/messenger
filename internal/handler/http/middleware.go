package http

import (
	"net/http"
	"strings"

	"messenger/internal/domain"
	"messenger/pkg/jwt"
)

func Auth(jwtManager *jwt.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := bearerUserID(r, jwtManager.ParseAccess)
		if err != nil {
			writeError(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), userID)))
	})
}

func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", domain.ErrUnauthorized
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", domain.ErrUnauthorized
	}

	return parts[1], nil
}

func bearerUserID(r *http.Request, parse func(string) (int64, error)) (int64, error) {
	token, err := bearerToken(r)
	if err != nil {
		return 0, err
	}

	userID, err := parse(token)
	if err != nil {
		return 0, domain.ErrUnauthorized
	}

	return userID, nil
}
