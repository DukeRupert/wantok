package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/realtime"
	"github.com/dukerupert/wantok/internal/store"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin allows connections from any origin (adjust for production)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket upgrades HTTP to WebSocket and registers the client.
// Route: GET /ws
func HandleWebSocket(hub *realtime.Hub, queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("websocket upgrade failed", "type", "request", "error", err)
			return
		}

		client := realtime.NewClient(hub, conn, user.ID, user.DisplayName)
		hub.Register(client)

		go client.WritePump()
		client.ReadPump() // Blocks until disconnect
	}
}
