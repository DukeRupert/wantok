package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
)

// AdminPageData holds data for the admin template.
type AdminPageData struct {
	User    *auth.User
	Users   []store.User
	Error   string
	Success string
}

// HandleAdminPage renders the admin user management page.
func HandleAdminPage(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		users, err := queries.ListUsers(ctx)
		if err != nil {
			slog.Error("failed to list users", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := AdminPageData{
			User:  user,
			Users: users,
		}

		if err := renderer.Render(w, "admin", data); err != nil {
			slog.Error("failed to render admin page", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleCreateUser processes the create user form.
func HandleCreateUser(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "type", "request", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		username := r.FormValue("username")
		displayName := r.FormValue("display_name")
		password := r.FormValue("password")
		isAdmin := r.FormValue("is_admin") == "on"

		// Validate input
		if username == "" || displayName == "" || password == "" {
			renderAdminError(w, queries, renderer, ctx, user, "All fields are required")
			return
		}

		// Hash password
		hash, err := auth.HashPassword(password)
		if err != nil {
			slog.Error("failed to hash password", "type", "request", "error", err)
			renderAdminError(w, queries, renderer, ctx, user, "Failed to create user")
			return
		}

		// Create user
		var adminFlag int64
		if isAdmin {
			adminFlag = 1
		}

		_, err = queries.CreateUser(ctx, store.CreateUserParams{
			Username:     username,
			DisplayName:  displayName,
			PasswordHash: hash,
			IsAdmin:      adminFlag,
		})
		if err != nil {
			slog.Error("failed to create user", "type", "request", "error", err)
			renderAdminError(w, queries, renderer, ctx, user, "Failed to create user (username may already exist)")
			return
		}

		slog.Info("user created", "type", "request", "username", username, "created_by", user.Username)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

// HandleUpdateUser processes the update user form.
func HandleUpdateUser(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		// Extract user ID from path
		userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "type", "request", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		displayName := r.FormValue("display_name")
		password := r.FormValue("password")
		isAdmin := r.FormValue("is_admin") == "on"

		// Validate input
		if displayName == "" {
			renderAdminError(w, queries, renderer, ctx, user, "Display name is required")
			return
		}

		// Get existing user to preserve password if not changed
		existingUser, err := queries.GetUserByID(ctx, userID)
		if err != nil {
			slog.Error("failed to get user", "type", "request", "error", err)
			renderAdminError(w, queries, renderer, ctx, user, "User not found")
			return
		}

		// Use existing password hash or generate new one
		passwordHash := existingUser.PasswordHash
		if password != "" {
			passwordHash, err = auth.HashPassword(password)
			if err != nil {
				slog.Error("failed to hash password", "type", "request", "error", err)
				renderAdminError(w, queries, renderer, ctx, user, "Failed to update user")
				return
			}
		}

		var adminFlag int64
		if isAdmin {
			adminFlag = 1
		}

		err = queries.UpdateUser(ctx, store.UpdateUserParams{
			ID:           userID,
			DisplayName:  displayName,
			PasswordHash: passwordHash,
			IsAdmin:      adminFlag,
		})
		if err != nil {
			slog.Error("failed to update user", "type", "request", "error", err)
			renderAdminError(w, queries, renderer, ctx, user, "Failed to update user")
			return
		}

		slog.Info("user updated", "type", "request", "user_id", userID, "updated_by", user.Username)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

// HandleDeleteUser deletes a user.
func HandleDeleteUser(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		// Extract user ID from path
		userID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Prevent self-deletion
		if userID == user.ID {
			http.Error(w, "Cannot delete yourself", http.StatusBadRequest)
			return
		}

		err = queries.DeleteUser(ctx, userID)
		if err != nil {
			slog.Error("failed to delete user", "type", "request", "error", err)
			http.Error(w, "Failed to delete user", http.StatusInternalServerError)
			return
		}

		slog.Info("user deleted", "type", "request", "user_id", userID, "deleted_by", user.Username)
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

// renderAdminError renders the admin page with an error message.
func renderAdminError(w http.ResponseWriter, queries *store.Queries, renderer *render.Renderer, ctx context.Context, user *auth.User, errMsg string) {
	users, _ := queries.ListUsers(ctx)

	data := AdminPageData{
		User:  user,
		Users: users,
		Error: errMsg,
	}

	w.WriteHeader(http.StatusBadRequest)
	renderer.Render(w, "admin", data)
}
