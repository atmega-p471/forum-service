package ws

import (
	"encoding/json"

	"github.com/forum/forum-service/internal/domain"
	"github.com/gorilla/websocket"
)

// Client represents a websocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *Hub) BroadcastMessage(message *domain.Message) {
	data, err := json.Marshal(message)
	if err != nil {
		return
	}
	h.broadcast <- data
}

// BroadcastMessages broadcasts multiple messages to all connected clients
func (h *Hub) BroadcastMessages(messages []*domain.Message) {
	data, err := json.Marshal(messages)
	if err != nil {
		return
	}
	h.broadcast <- data
}
