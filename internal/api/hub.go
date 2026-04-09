package api

import (
	"sync"
)

// VoteUpdate es lo que el móvil recibirá por el socket
type VoteUpdate struct {
	PollID   string `json:"poll_id"`
	OptionID string `json:"option_id"`
	NewCount int    `json:"new_count"`
}

type SocketMessage struct {
    Event   string      `json:"-"`       // "poll_created", "vote_cast", etc.
    Payload interface{} `json:"payload"` // El objeto (PollOutput o VoteUpdate)
}

type Hub struct {
	// Canales de comunicación
	Broadcast  chan SocketMessage
	Register   chan chan SocketMessage
	Unregister chan chan SocketMessage
	clients    map[chan SocketMessage]bool
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan SocketMessage),
		Register:   make(chan chan SocketMessage),
		Unregister: make(chan chan SocketMessage),
		clients:    make(map[chan SocketMessage]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.Unregister:
			h.mu.Lock()
			delete(h.clients, client)
			close(client)
			h.mu.Unlock()
		case update := <-h.Broadcast:
			h.mu.Lock()
			for client := range h.clients {
				client <- update
			}
			h.mu.Unlock()
		}
	}
}