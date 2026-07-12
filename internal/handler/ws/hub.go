package ws

import (
	"context"
	"sync"
)

type Hub struct {
	mu    sync.RWMutex
	users map[int64]map[*Client]struct{}
	done  chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		users: make(map[int64]map[*Client]struct{}),
		done:  make(chan struct{}),
	}
}

// Register adds a client connection. Returns true if this is the user's first active connection.
func (h *Hub) Register(client *Client) (becameOnline bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.users[client.userID]
	if !ok {
		conns = make(map[*Client]struct{})
		h.users[client.userID] = conns
		becameOnline = true
	}
	conns[client] = struct{}{}
	return becameOnline
}

// Unregister removes a client connection. Returns true if this was the user's last connection.
func (h *Hub) Unregister(client *Client) (becameOffline bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns, ok := h.users[client.userID]
	if !ok {
		return false
	}
	delete(conns, client)
	if len(conns) == 0 {
		delete(h.users, client.userID)
		return true
	}
	return false
}

func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.users[userID]
	return ok && len(conns) > 0
}

func (h *Hub) BroadcastNewMessage(chatID int64, senderID int64, payload []byte, recipientIDs []int64) {
	h.broadcastExcept(senderID, payload, recipientIDs)
}

func (h *Hub) BroadcastRead(chatID int64, readerID int64, payload []byte, recipientIDs []int64) {
	h.broadcastExcept(readerID, payload, recipientIDs)
}

func (h *Hub) BroadcastChatUpdated(chatID int64, actorUserID int64, payload []byte, recipientIDs []int64) {
	h.broadcastExcept(actorUserID, payload, recipientIDs)
}

func (h *Hub) BroadcastToUsers(payload []byte, recipientIDs []int64) {
	h.broadcastExcept(0, payload, recipientIDs)
}

func (h *Hub) broadcastExcept(excludeUserID int64, payload []byte, recipientIDs []int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, userID := range recipientIDs {
		if excludeUserID != 0 && userID == excludeUserID {
			continue
		}
		conns := h.users[userID]
		for client := range conns {
			_ = client.enqueue(payload)
		}
	}
}

func (h *Hub) Shutdown(ctx context.Context) {
	close(h.done)

	h.mu.Lock()
	clients := make([]*Client, 0)
	for _, conns := range h.users {
		for client := range conns {
			clients = append(clients, client)
		}
	}
	h.mu.Unlock()

	for _, client := range clients {
		client.close()
	}

	if len(clients) == 0 {
		return
	}

	done := make(chan struct{})
	go func() {
		for _, client := range clients {
			client.waitClosed()
		}
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (h *Hub) Done() <-chan struct{} {
	return h.done
}
