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
//
// Implementation notes:
//   - Validate authentication via session cookie (same as RequireAuth)
//   - Upgrade connection to WebSocket
//   - Create Client with user info
//   - Register client with hub
//   - Start ReadPump and WritePump goroutines
//   - ReadPump handles cleanup on disconnect
func HandleWebSocket(hub *realtime.Hub, queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement
		// 1. Get user from context (requires auth middleware)
		ctx := r.Context()
		user := auth.GetUser(ctx)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Upgrade to WebSocket
		// conn, err := upgrader.Upgrade(w, r, nil)
		// if err != nil {
		//     slog.Error("websocket upgrade failed", "type", "request", "error", err)
		//     return
		// }

		// 3. Create client
		// client := realtime.NewClient(hub, conn, user.ID, user.DisplayName)

		// 4. Register with hub
		// hub.Register(client)

		// 5. Start pumps (WritePump in goroutine, ReadPump blocks)
		// go client.WritePump()
		// client.ReadPump() // Blocks until disconnect

		slog.Info("websocket connection established", "type", "request", "user_id", user.ID)
		http.Error(w, "Not implemented", http.StatusNotImplemented)
	}
}
