package http

import (
	"net/http"

	"messenger/internal/domain"
)

type addMemberRequest struct {
	UserID int64 `json:"user_id"`
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	var req addMemberRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}
	if req.UserID <= 0 {
		writeError(w, domain.ErrValidation)
		return
	}

	if err := h.svc.AddMember(r.Context(), callerID, chatID, req.UserID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	userID, ok := parsePathInt64(w, r, "user_id")
	if !ok {
		return
	}

	if err := h.svc.RemoveMember(r.Context(), callerID, chatID, userID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
