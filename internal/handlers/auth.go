package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
)

const (
	sessionCookieName = "session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// LoginPageData holds data for the login template.
type LoginPageData struct {
	Error string
}

// HandleLoginPage renders the login form.
// Redirects to / if user is already authenticated.
func HandleLoginPage(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Check if user is already authenticated
		user := auth.GetUser(ctx)
		// If authenticated, redirect to /
		if user != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// Render login template
		if err := renderer.Render(w, "login", LoginPageData{}); err != nil {
			slog.Error("failed to render login page", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleLogin processes the login form submission.
// On success: creates session, sets cookie, redirects to /
// On failure: re-renders login page with error
func HandleLogin(queries *store.Queries, renderer *render.Renderer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Parse form to get username and password
		if err := r.ParseForm(); err != nil {
			slog.Error("ParseForm() error", "error", err)
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		// Look up user by username
		user, err := queries.GetUserByUsername(ctx, username)
		// If not found, re-render login with error (don't reveal user doesn't exist)
		if err != nil {
			slog.Error("GetUserByUsername() error", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			renderer.Render(w, "login", LoginPageData{Error: "Invalid username or password"})
			return
		}
		// Check password
		isValid := auth.CheckPassword(user.PasswordHash, password)
		// If password wrong, re-render login with error
		if !isValid {
			slog.Info("Invalid password")
			w.WriteHeader(http.StatusUnauthorized)
			renderer.Render(w, "login", LoginPageData{Error: "Invalid username or password"})
			return
		}
		// Create session
		token, err := auth.CreateSession(ctx, queries, user.ID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, token)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// HandleLogout clears the session and redirects to login.
func HandleLogout(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// TODO: Get session token from sessionCookie
		sessionCookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			slog.Info("no cookie found", "error", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		// TODO: Delete session from database using auth.DeleteSession
		token := sessionCookie.Value
		err = auth.DeleteSession(ctx, queries, token)
		if err != nil {
			slog.Error("failed to delete session from database", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		clearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// setSessionCookie sets the session cookie with secure defaults.
func setSessionCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// clearSessionCookie removes the session cookie from the browser.
func clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
