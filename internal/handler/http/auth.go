package http

import (
	"net/http"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	user, err := h.svc.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, registerResponse{
		ID:    user.ID,
		Login: user.Login,
	})
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	access, refresh, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	})
}

type refreshResponse struct {
	AccessToken string `json:"access_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	token, err := bearerToken(r)
	if err != nil {
		writeError(w, err)
		return
	}

	access, err := h.svc.Refresh(r.Context(), token)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, refreshResponse{AccessToken: access})
}
