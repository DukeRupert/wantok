package handlers

import (
	"log/slog"
	"net/http"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/store"
	"github.com/dukerupert/wantok/internal/views/pages"
)

const (
	sessionCookieName = "session"
	sessionMaxAge     = 30 * 24 * 60 * 60 // 30 days in seconds
	maxInputLength    = 256               // Max length for login inputs
)

// SecureCookies controls whether Secure flag is set on cookies.
// Set to false for development over HTTP.
var SecureCookies = true

// HandleLoginPage renders the login form.
// Redirects to / if user is already authenticated.
func HandleLoginPage(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Check if user is already authenticated
		if token, err := r.Cookie(sessionCookieName); err == nil {
			if row, err := auth.ValidateSession(ctx, queries, token.Value); err == nil && row != nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		// Render login template
		if err := pages.Login(pages.LoginPageData{}).Render(ctx, w); err != nil {
			slog.Error("failed to render login page", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleLogin processes the login form submission.
// On success: creates session, sets cookie, redirects to /
// On failure: re-renders login page with error
func HandleLogin(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// Parse form to get username and password
		if err := r.ParseForm(); err != nil {
			slog.Error("ParseForm() error", "type", "request", "error", err)
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Basic length validation to prevent abuse
		if len(username) > maxInputLength || len(password) > maxInputLength {
			w.WriteHeader(http.StatusBadRequest)
			pages.Login(pages.LoginPageData{Error: "Invalid username or password"}).Render(ctx, w)
			return
		}

		// Look up user by username
		user, err := queries.GetUserByUsername(ctx, username)
		// If not found, re-render login with error (don't reveal user doesn't exist)
		if err != nil {
			slog.Error("GetUserByUsername() error", "type", "request", "error", err)
			w.WriteHeader(http.StatusUnauthorized)
			pages.Login(pages.LoginPageData{Error: "Invalid username or password"}).Render(ctx, w)
			return
		}
		// Check password
		isValid := auth.CheckPassword(user.PasswordHash, password)
		// If password wrong, re-render login with error
		if !isValid {
			slog.Info("Invalid password", "type", "request")
			w.WriteHeader(http.StatusUnauthorized)
			pages.Login(pages.LoginPageData{Error: "Invalid username or password"}).Render(ctx, w)
			return
		}
		// Create session
		token, err := auth.CreateSession(ctx, queries, user.ID)
		if err != nil {
			slog.Error("failed to create session", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		setSessionCookie(w, token)
		slog.Info("user logged in", "type", "request", "username", user.Username, "user_id", user.ID)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// HandleHome renders the home page for authenticated users.
func HandleHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		if user == nil {
			w.Write([]byte("Welcome to Wantok"))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Wantok</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen flex items-center justify-center">
	<div class="text-center">
		<h1 class="text-2xl font-bold text-gray-800 mb-4">Welcome, ` + user.DisplayName + `</h1>
		<form action="/auth/logout" method="POST">
			<button type="submit" class="px-4 py-2 bg-emerald-600 hover:bg-emerald-700 text-white rounded-md transition duration-200">
				Logout
			</button>
		</form>
	</div>
</body>
</html>`
		w.Write([]byte(html))
	}
}

// HandleLogout clears the session and redirects to login.
func HandleLogout(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// TODO: Get session token from sessionCookie
		sessionCookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			slog.Info("no cookie found", "type", "request", "error", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		// TODO: Delete session from database using auth.DeleteSession
		token := sessionCookie.Value
		err = auth.DeleteSession(ctx, queries, token)
		if err != nil {
			slog.Error("failed to delete session from database", "type", "request", "error", err)
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
		Secure:   SecureCookies,
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
		Secure:   SecureCookies,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}
