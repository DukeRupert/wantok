package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Message represents a WebSocket message to be sent to clients.
type Message struct {
	Type    string `json:"type"`    // "message", "typing", "presence", etc.
	Payload any    `json:"payload"` // Message content
}

// Hub maintains the set of active clients and broadcasts messages to them.
// It uses a single goroutine to handle all register/unregister/broadcast operations
// to avoid race conditions on the clients map.
type Hub struct {
	// clients maps user ID to their connected clients (supports multiple devices)
	clients map[int64]map[*Client]bool

	// register channel for new client connections
	register chan *Client

	// unregister channel for client disconnections
	unregister chan *Client

	// broadcast channel for messages to specific users
	broadcast chan *UserMessage

	// mu protects clients map for read operations outside Run()
	mu sync.RWMutex
}

// UserMessage wraps a serialized message with target user ID.
type UserMessage struct {
	UserID int64
	Data   []byte
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[int64]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *UserMessage, 256), // buffered to prevent blocking
	}
}

// Run starts the hub's main loop. Should be called in a goroutine.
// Handles all client registration, unregistration, and message broadcasting.
func (h *Hub) Run() {
	slog.Info("hub started", "type", "lifecycle")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			slog.Info("client connected", "type", "websocket", "user_id", client.UserID, "display_name", client.DisplayName)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					client.Close()
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.Unlock()
			slog.Info("client disconnected", "type", "websocket", "user_id", client.UserID, "display_name", client.DisplayName)

		case userMsg := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[userMsg.UserID]
			h.mu.RUnlock()

			for client := range clients {
				if !client.Send(userMsg.Data) {
					// Buffer full, disconnect client
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
		}
	}
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// SendToUser sends a message to all connected clients for a user.
// Used by message handlers to broadcast new messages.
// Non-blocking: if user has no clients or channel is full, message is dropped.
func (h *Hub) SendToUser(userID int64, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal message", "type", "websocket", "error", err)
		return
	}

	select {
	case h.broadcast <- &UserMessage{UserID: userID, Data: data}:
	default:
		slog.Warn("broadcast channel full, dropping message", "type", "websocket", "user_id", userID)
	}
}

// ClientCount returns the number of connected clients for a user.
// Useful for presence features.
func (h *Hub) ClientCount(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID])
}

// IsOnline returns whether a user has any connected clients.
func (h *Hub) IsOnline(userID int64) bool {
	return h.ClientCount(userID) > 0
}
