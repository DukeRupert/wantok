package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/store"
)

// UserResponse is the JSON response for a user.
type UserResponse struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

// HandleListUsers returns all users except the current user.
// Used for starting new conversations.
func HandleListUsers(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		users, err := queries.ListUsersExcept(ctx, user.ID)
		if err != nil {
			slog.Error("failed to list users", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Convert to response format (exclude sensitive fields)
		response := make([]UserResponse, len(users))
		for i, u := range users {
			response[i] = UserResponse{
				ID:          u.ID,
				Username:    u.Username,
				DisplayName: u.DisplayName,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			slog.Error("failed to encode users", "type", "request", "error", err)
		}
	}
}
