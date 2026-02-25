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

type Hub struct {
	// Canales de comunicación
	Broadcast  chan VoteUpdate
	Register   chan chan VoteUpdate
	Unregister chan chan VoteUpdate
	clients    map[chan VoteUpdate]bool
	mu         sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Broadcast:  make(chan VoteUpdate),
		Register:   make(chan chan VoteUpdate),
		Unregister: make(chan chan VoteUpdate),
		clients:    make(map[chan VoteUpdate]bool),
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