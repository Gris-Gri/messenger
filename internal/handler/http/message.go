package http

import (
	"net/http"

	"messenger/internal/domain"
)

type reactionCountsResponse struct {
	Like       int     `json:"like"`
	Dislike    int     `json:"dislike"`
	Heart      int     `json:"heart"`
	MyReaction *string `json:"my_reaction"`
}

type messageResponse struct {
	ID        int64                   `json:"id"`
	SenderID  int64                   `json:"sender_id"`
	Body      string                  `json:"body"`
	CreatedAt string                  `json:"created_at"`
	EditedAt  *string                 `json:"edited_at,omitempty"`
	Reactions *reactionCountsResponse `json:"reactions,omitempty"`
}

type editMessageRequest struct {
	Body string `json:"body"`
}

type setReactionRequest struct {
	Reaction string `json:"reaction"`
}

func toMessageResponse(msg domain.Message) messageResponse {
	resp := messageResponse{
		ID:        msg.ID,
		SenderID:  msg.SenderID,
		Body:      msg.Body,
		CreatedAt: msg.CreatedAt.UTC().Format(timeRFC3339Nano),
	}
	if msg.EditedAt != nil {
		formatted := msg.EditedAt.UTC().Format(timeRFC3339Nano)
		resp.EditedAt = &formatted
	}
	return resp
}

func toMessageWithReactionsResponse(msg domain.MessageWithReactions) messageResponse {
	resp := toMessageResponse(msg.Message)
	reactions := toReactionCountsResponse(msg.Reactions)
	resp.Reactions = &reactions
	return resp
}

func toReactionCountsResponse(summary domain.ReactionSummary) reactionCountsResponse {
	return reactionCountsResponse{
		Like:       summary.Like,
		Dislike:    summary.Dislike,
		Heart:      summary.Heart,
		MyReaction: summary.MyReaction,
	}
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
		resp = append(resp, toMessageWithReactionsResponse(msg))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) EditMessage(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	messageID, ok := parsePathInt64(w, r, "message_id")
	if !ok {
		return
	}

	var req editMessageRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	msg, err := h.svc.EditMessage(r.Context(), callerID, chatID, messageID, req.Body)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toMessageResponse(*msg))
}

func (h *Handler) SetMessageReaction(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.callerID(r)
	if !ok {
		writeError(w, domain.ErrUnauthorized)
		return
	}

	chatID, ok := parsePathInt64(w, r, "id")
	if !ok {
		return
	}

	messageID, ok := parsePathInt64(w, r, "message_id")
	if !ok {
		return
	}

	var req setReactionRequest
	if !h.decodeJSON(w, r, &req) {
		return
	}

	summary, err := h.svc.SetMessageReaction(r.Context(), callerID, chatID, messageID, req.Reaction)
	if err != nil {
		writeError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toReactionCountsResponse(summary))
}
