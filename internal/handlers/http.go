package handlers

import (
	"net/http"

	"github.com/dukerupert/wantok/internal/store"
)

func NewServer(queries *store.Queries) http.Handler {
	mux := http.NewServeMux()
	// routes

	// global middleware

	return mux
}