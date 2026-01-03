package realtime

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	mu       sync.RWMutex
	clients  map[int64]map[*Client]struct{}
	presence map[int64]map[int64]int
}

type Client struct {
	TenantID int64
	UserID   int64
	Send     chan []byte
}

func NewHub() *Hub {
	return &Hub{
		clients:  map[int64]map[*Client]struct{}{},
		presence: map[int64]map[int64]int{},
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.TenantID] == nil {
		h.clients[client.TenantID] = map[*Client]struct{}{}
	}
	h.clients[client.TenantID][client] = struct{}{}
	if h.presence[client.TenantID] == nil {
		h.presence[client.TenantID] = map[int64]int{}
	}
	h.presence[client.TenantID][client.UserID]++
	h.broadcastPresenceLocked(client.TenantID)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[client.TenantID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.clients, client.TenantID)
		}
	}
	if presence, ok := h.presence[client.TenantID]; ok {
		if count, ok := presence[client.UserID]; ok {
			if count <= 1 {
				delete(presence, client.UserID)
			} else {
				presence[client.UserID] = count - 1
			}
		}
		h.broadcastPresenceLocked(client.TenantID)
	}
	close(client.Send)
}

func (h *Hub) Broadcast(tenantID int64, payload any) {
	message, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := h.clients[tenantID]
	h.mu.RUnlock()
	for client := range clients {
		select {
		case client.Send <- message:
		default:
		}
	}
}

func (h *Hub) broadcastPresenceLocked(tenantID int64) {
	presence := h.presence[tenantID]
	users := make([]int64, 0, len(presence))
	for userID := range presence {
		users = append(users, userID)
	}
	payload := map[string]any{
		"type":  "presence.update",
		"users": users,
	}
	message, err := json.Marshal(payload)
	if err != nil {
		return
	}
	for client := range h.clients[tenantID] {
		select {
		case client.Send <- message:
		default:
		}
	}
}
