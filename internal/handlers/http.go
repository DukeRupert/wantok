package handlers

import (
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
)

func NewServer(queries *store.Queries, renderer *render.Renderer) http.Handler {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("GET /login", HandleLoginPage(queries, renderer))
	mux.HandleFunc("POST /auth/login", HandleLogin(queries, renderer))
	mux.HandleFunc("POST /auth/logout", HandleLogout(queries))

	// Protected routes
	mux.Handle("GET /", auth.RequireAuth(queries)(HandleHome()))

	// TODO: Admin routes (wrap with auth.RequireAdmin)

	return mux
}