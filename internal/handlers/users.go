package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/store"
	"github.com/dukerupert/wantok/internal/views/partials"
)

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

		// Convert to template format
		userList := make([]partials.UserListItem, len(users))
		for i, u := range users {
			userList[i] = partials.UserListItem{
				ID:          u.ID,
				DisplayName: u.DisplayName,
			}
		}

		if err := partials.UserList(userList).Render(ctx, w); err != nil {
			slog.Error("failed to render user list", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}
