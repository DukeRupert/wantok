package realtime

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

// Client represents a single WebSocket connection.
// Each browser tab/device creates a new Client.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// UserID of the authenticated user.
	UserID int64

	// DisplayName for logging/debugging.
	DisplayName string
}

// NewClient creates a new Client instance.
func NewClient(hub *Hub, conn *websocket.Conn, userID int64, displayName string) *Client {
	return &Client{
		hub:         hub,
		conn:        conn,
		send:        make(chan []byte, 256),
		UserID:      userID,
		DisplayName: displayName,
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
// Runs in a dedicated goroutine for each client.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// Connection closed or error - exit loop
			break
		}
		// Future: process incoming message (typing, read receipt, etc.)
	}
}

// WritePump pumps messages from the hub to the websocket connection.
// Runs in a dedicated goroutine for each client.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send queues a message to be sent to this client.
// Returns false if the send buffer is full (client will be disconnected).
func (c *Client) Send(data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		return false
	}
}

// Close closes the client's send channel.
// Should only be called by the hub.
func (c *Client) Close() {
	close(c.send)
}
