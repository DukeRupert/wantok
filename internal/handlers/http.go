package handlers

import (
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/realtime"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
)

func NewServer(queries *store.Queries, renderer *render.Renderer, hub *realtime.Hub) http.Handler {
	mux := http.NewServeMux()

	// Auth routes (public)
	mux.HandleFunc("GET /login", HandleLoginPage(queries, renderer))
	mux.HandleFunc("POST /auth/login", HandleLogin(queries, renderer))
	mux.HandleFunc("POST /auth/logout", HandleLogout(queries))

	// Protected routes (require auth)
	mux.Handle("GET /", auth.RequireAuth(queries)(HandleChatPage(queries, renderer)))
	mux.Handle("GET /users", auth.RequireAuth(queries)(HandleListUsers(queries)))

	// Messaging routes (require auth)
	mux.Handle("GET /conversations", auth.RequireAuth(queries)(HandleGetConversations(queries)))
	mux.Handle("GET /conversations/{userID}/messages", auth.RequireAuth(queries)(HandleGetMessages(queries)))
	mux.Handle("POST /conversations/{userID}/messages", auth.RequireAuth(queries)(HandleSendMessage(queries)))

	// Admin routes (require auth + admin)
	mux.Handle("GET /admin", auth.RequireAuth(queries)(auth.RequireAdmin(HandleAdminPage(queries, renderer))))
	mux.Handle("POST /admin/users", auth.RequireAuth(queries)(auth.RequireAdmin(HandleCreateUser(queries, renderer))))
	mux.Handle("POST /admin/users/{id}", auth.RequireAuth(queries)(auth.RequireAdmin(HandleUpdateUser(queries, renderer))))
	mux.Handle("POST /admin/users/{id}/delete", auth.RequireAuth(queries)(auth.RequireAdmin(HandleDeleteUser(queries))))

	// WebSocket route (require auth)
	mux.Handle("GET /ws", auth.RequireAuth(queries)(HandleWebSocket(hub, queries)))

	return mux
}