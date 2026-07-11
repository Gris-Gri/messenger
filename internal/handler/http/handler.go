package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"messenger/internal/domain"
	"messenger/internal/service"
	"messenger/pkg/jwt"
)

const defaultMessageLimit = 50

type Handler struct {
	svc *service.Service
	jwt *jwt.Manager
}

func NewHandler(svc *service.Service, jwtManager *jwt.Manager) *Handler {
	return &Handler{svc: svc, jwt: jwtManager}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, domain.ErrValidation)
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeError(w, domain.ErrValidation)
		return false
	}
	return true
}

func (h *Handler) callerID(r *http.Request) (int64, bool) {
	return userIDFromContext(r.Context())
}

func parsePathInt64(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	v, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || v <= 0 {
		writeError(w, domain.ErrValidation)
		return 0, false
	}
	return v, true
}

func parseQueryInt64(w http.ResponseWriter, r *http.Request, name string, defaultVal int64) (int64, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(w, domain.ErrValidation)
		return 0, false
	}
	return v, true
}

func parseQueryInt(w http.ResponseWriter, r *http.Request, name string, defaultVal int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultVal, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		writeError(w, domain.ErrValidation)
		return 0, false
	}
	return v, true
}
