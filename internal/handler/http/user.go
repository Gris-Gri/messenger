package http

import (
	"net/http"
	"strings"

	"messenger/internal/domain"
)

const defaultUserSearchLimit = 20

type userSearchResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func (h *Handler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	login := strings.TrimSpace(r.URL.Query().Get("login"))
	limit, ok := parseQueryInt(w, r, "limit", defaultUserSearchLimit)
	if !ok {
		return
	}

	users, err := h.svc.SearchUsers(r.Context(), callerID, login, limit)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := make([]userSearchResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, userSearchResponse{
			ID:    u.ID,
			Login: u.Login,
		})
	}

	h.writeJSON(w, http.StatusOK, resp)
}
