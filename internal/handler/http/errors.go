package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"messenger/internal/domain"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

func writeError(w http.ResponseWriter, err error) {
	status, body := mapError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: body})
}

func mapError(err error) (int, errorBody) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest, errorBody{Code: "validation_error", Message: err.Error()}
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, errorBody{Code: "invalid_credentials", Message: err.Error()}
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, errorBody{Code: "unauthorized", Message: err.Error()}
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, errorBody{Code: "forbidden", Message: err.Error()}
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, errorBody{Code: "not_found", Message: err.Error()}
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, errorBody{Code: "conflict", Message: err.Error()}
	default:
		return http.StatusInternalServerError, errorBody{Code: "internal_error", Message: "Внутренняя ошибка сервера"}
	}
}
