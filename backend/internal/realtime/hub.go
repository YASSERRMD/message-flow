package realtime

import (
	"encoding/json"
	"sync"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[int64]map[*Client]struct{}
}

type Client struct {
	TenantID int64
	Send     chan []byte
}

func NewHub() *Hub {
	return &Hub{clients: map[int64]map[*Client]struct{}{}}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[client.TenantID] == nil {
		h.clients[client.TenantID] = map[*Client]struct{}{}
	}
	h.clients[client.TenantID][client] = struct{}{}
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
