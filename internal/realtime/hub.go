package realtime

import (
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

// UserMessage wraps a message with target user ID.
type UserMessage struct {
	UserID  int64
	Message *Message
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
//
// Implementation notes:
//   - Use select to handle register, unregister, and broadcast channels
//   - For register: add client to clients[userID] map
//   - For unregister: remove client, close send channel, cleanup empty user maps
//   - For broadcast: send message to all clients for the target user
func (h *Hub) Run() {
	// TODO: Implement
	// for {
	//     select {
	//     case client := <-h.register:
	//         // Add client to clients map
	//         // Create user's client set if doesn't exist
	//         // Log connection
	//
	//     case client := <-h.unregister:
	//         // Remove client from clients map
	//         // Close client's send channel
	//         // Delete user's map if empty
	//         // Log disconnection
	//
	//     case userMsg := <-h.broadcast:
	//         // Get all clients for target user
	//         // Send message to each client's send channel
	//         // If send blocks (buffer full), unregister client
	//     }
	// }
	slog.Info("hub started", "type", "lifecycle")
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
//
// Implementation notes:
//   - Called from HTTP handlers after message creation
//   - Should send to both sender (other devices) and recipient
//   - Non-blocking: if user has no clients, message is dropped
func (h *Hub) SendToUser(userID int64, msg *Message) {
	// TODO: Implement
	// Send to broadcast channel
	// select {
	// case h.broadcast <- &UserMessage{UserID: userID, Message: msg}:
	// default:
	//     // Channel full, log warning
	// }
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
