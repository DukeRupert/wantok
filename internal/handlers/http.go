package handlers

import (
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/realtime"
	"github.com/dukerupert/wantok/internal/store"
)

func NewServer(queries *store.Queries, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// Auth routes (public)
	mux.HandleFunc("GET /login", HandleLoginPage(queries))
	mux.HandleFunc("POST /auth/login", HandleLogin(queries))
	mux.HandleFunc("POST /auth/logout", HandleLogout(queries))

	// Protected routes (require auth)
	mux.Handle("GET /", auth.RequireAuth(queries)(HandleChatPage(queries)))
	mux.Handle("GET /users", auth.RequireAuth(queries)(HandleListUsers(queries)))

	// Messaging routes (require auth)
	mux.Handle("GET /conversations", auth.RequireAuth(queries)(HandleGetConversations(queries)))
	mux.Handle("GET /conversations/{userID}/messages", auth.RequireAuth(queries)(HandleGetMessages(queries)))
	mux.Handle("POST /conversations/{userID}/messages", auth.RequireAuth(queries)(HandleSendMessage(queries, hub)))

	// Admin routes (require auth + admin)
	mux.Handle("GET /admin", auth.RequireAuth(queries)(auth.RequireAdmin(HandleAdminPage(queries))))
	mux.Handle("POST /admin/users", auth.RequireAuth(queries)(auth.RequireAdmin(HandleCreateUser(queries))))
	mux.Handle("POST /admin/users/{id}", auth.RequireAuth(queries)(auth.RequireAdmin(HandleUpdateUser(queries))))
	mux.Handle("POST /admin/users/{id}/delete", auth.RequireAuth(queries)(auth.RequireAdmin(HandleDeleteUser(queries))))

	// WebSocket route (require auth)
	mux.Handle("GET /ws", auth.RequireAuth(queries)(HandleWebSocket(hub, queries)))

	return mux
}
