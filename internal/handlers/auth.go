package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/store"
)

const (
	sessionCookieName = "session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
)

// HandleLoginPage renders the login form.
// Redirects to / if user is already authenticated.
func HandleLoginPage(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// TODO: Check if user is already authenticated via auth.GetUser
		user := auth.GetUser(ctx)
		// TODO: If authenticated, redirect to /
		if user != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		// TODO: Render login template
		w.Write([]byte("Login page"))
	}
}

// HandleLogin processes the login form submission.
// On success: creates session, sets cookie, redirects to /
// On failure: re-renders login page with error
func HandleLogin(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// TODO: Parse form to get username and password
		if err := r.ParseForm(); err != nil {
			slog.Error("ParseForm() error", "error", err)
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")
		// TODO: Look up user by username using queries.GetUserByUsername
		user, err := queries.GetUserByUsername(ctx, username)
		// TODO: If not found, re-render login with error (don't reveal user doesn't exist)
		if err != nil {
			slog.Error("GetUserByUsername() error", "error", err)
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		// TODO: Check password using auth.CheckPassword
		isValid := auth.CheckPassword(user.PasswordHash, password)
		// TODO: If password wrong, re-render login with error
		if !isValid {
			slog.Info("Invalid password")
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}
		// TODO: Create session using auth.CreateSession
		token, err := auth.CreateSession(ctx, queries, user.ID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		// TODO: Set cookie with token (Name: sessionCookieName, HttpOnly, SameSite=Lax, Path=/, MaxAge=sessionMaxAge)
		cookie := http.Cookie{
			Name: sessionCookieName,
			Value: token,
			Path: "/",
			MaxAge: sessionMaxAge,
			HttpOnly: true,
			Secure: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)
		// TODO: Redirect to /
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// HandleLogout clears the session and redirects to login.
func HandleLogout(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get session token from cookie
		// TODO: Delete session from database using auth.DeleteSession
		// TODO: Clear cookie (set MaxAge -1)
		// TODO: Redirect to /login
	}
}

// setSessionCookie sets the session cookie with secure defaults.
func setSessionCookie(w http.ResponseWriter, token string) {
	// TODO: Create http.Cookie with:
	//   - Name: sessionCookieName
	//   - Value: token
	//   - Path: /
	//   - MaxAge: sessionMaxAge
	//   - HttpOnly: true
	//   - SameSite: http.SameSiteLaxMode
	// TODO: Call http.SetCookie
}

// clearSessionCookie removes the session cookie from the browser.
func clearSessionCookie(w http.ResponseWriter) {
	// TODO: Create http.Cookie with empty value and MaxAge -1
	// TODO: Call http.SetCookie
}