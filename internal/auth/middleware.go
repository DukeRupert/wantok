package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/store"
)

// to avoid collisions.
type contextKey string

const (
	userContextKey    contextKey = "user"
	sessionCookieName            = "session"
)

// User represents the authenticated user stored in context.
// Extracted from store.GetSessionWithUserRow for cleaner access.
type User struct {
	ID          int64
	Username    string
	DisplayName string
	IsAdmin     bool
}

// RequireAuth is middleware that validates the session cookie.
// Redirects to /login if not authenticated.
// Stores User in request context on success.
func RequireAuth(queries *store.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			token, err := r.Cookie(sessionCookieName)
			// If missing, redirect to /login
			if err != nil {
				if err == http.ErrNoCookie {
					slog.Info("session cookie not found. redirect to /login")
					http.Redirect(w, r, "/login", http.StatusSeeOther)
					return
				}
				slog.Error("failed to read session cookie", "error", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			// Call ValidateSession with token
			row, err := ValidateSession(ctx, queries, token.Value)
			// If invalid/expired, clear cookie and redirect to /login
			if err != nil {
				slog.Info("invalid session cookie", "error", err)
				clearSessionCookie(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			// Create User, store in context, and call next handler
			user := User{
				ID:          row.UserID,
				Username:    row.Username,
				DisplayName: row.DisplayName,
				IsAdmin:     row.IsAdmin != 0,
			}
			ctxWithUser := context.WithValue(ctx, userContextKey, &user)
			req := r.WithContext(ctxWithUser)
			next.ServeHTTP(w, req)
		})
	}
}

// RequireAdmin is middleware that ensures the user is an admin.
// Must be used after RequireAuth.
// Returns 403 Forbidden if user is not an admin.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := GetUser(ctx)
		// If user is nil, return 401 (RequireAuth wasn't called)
		if user == nil {
			http.Error(w, "Unauthenticated request", http.StatusUnauthorized)
			return
		}
		// If not admin, return 403 Forbidden
		if !user.IsAdmin {
			http.Error(w, "Unauthorized request", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetUser extracts the authenticated user from the request context.
// Returns nil if no user is present (request not authenticated).
func GetUser(ctx context.Context) *User {
	value := ctx.Value(userContextKey)
	user, ok := value.(*User); if !ok {
		return nil
	}
	return user
}

// clearSessionCookie sets an expired cookie to clear it from the browser.
func clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name: string(sessionCookieName),
		Value: "",
		Path: "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}
