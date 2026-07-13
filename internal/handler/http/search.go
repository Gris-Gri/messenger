package http

import (
	"net/http"
	"strings"

	"messenger/internal/domain"
)

func (h *Handler) SearchMessages(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, domain.ErrValidation)
		return
	}

	messages, err := h.svc.Search(r.Context(), callerID, chatID, query)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]messageResponse, 0, len(messages))
	for _, msg := range messages {
		resp = append(resp, toMessageResponse(msg))
	}

	h.writeJSON(w, http.StatusOK, resp)
}
