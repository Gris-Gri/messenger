package http

import (
	"net/http"

	"messenger/internal/domain"
)

type messageResponse struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"sender_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	beforeID, ok := parseQueryInt64(w, r, "before_id", 0)
	if !ok {
		return
	}

	limit, ok := parseQueryInt(w, r, "limit", defaultMessageLimit)
	if !ok {
		return
	}

	messages, err := h.svc.GetMessageHistory(r.Context(), callerID, chatID, beforeID, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]messageResponse, 0, len(messages))
	for _, msg := range messages {
		resp = append(resp, messageResponse{
			ID:        msg.ID,
			SenderID:  msg.SenderID,
			Body:      msg.Body,
			CreatedAt: msg.CreatedAt.UTC().Format(timeRFC3339Nano),
		})
	}

	h.writeJSON(w, http.StatusOK, resp)
}
