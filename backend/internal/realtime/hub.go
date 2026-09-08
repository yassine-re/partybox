package realtime

import (
	"encoding/json"
	"errors"
	"sync"
)

var ErrHubClosed = errors.New("realtime hub is closed")

// Hub partitions clients by game ID. Broadcast never blocks a mutation HTTP
// handler: a client that cannot keep up is disconnected and will reconnect.
type Hub struct {
	mu     sync.RWMutex
	games  map[string]map[*Client]struct{}
	closed bool
}

func NewHub() *Hub {
	return &Hub{games: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrHubClosed
	}
	clients := h.games[client.gameID]
	if clients == nil {
		clients = make(map[*Client]struct{})
		h.games[client.gameID] = clients
	}
	clients[client] = struct{}{}
	return nil
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.games[client.gameID]
	delete(clients, client)
	if len(clients) == 0 {
		delete(h.games, client.gameID)
	}
}

func (h *Hub) Broadcast(event Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.games[event.GameID]))
	for client := range h.games[event.GameID] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.enqueue(payload)
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	clients := make([]*Client, 0)
	for _, gameClients := range h.games {
		for client := range gameClients {
			clients = append(clients, client)
		}
	}
	h.games = make(map[string]map[*Client]struct{})
	h.mu.Unlock()
	for _, client := range clients {
		client.close()
	}
}

func (h *Hub) connectionCount(gameID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.games[gameID])
}
