package handlers

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/email"
	"github.com/dukerupert/wantok/internal/store"
	"github.com/dukerupert/wantok/internal/validate"
	"github.com/dukerupert/wantok/internal/views/pages"
)

const (
	magicLinkExpiryHours   = 24
	maxMagicLinksPerHour   = 3
)

// HandleMagicLinkPage renders the magic link request form.
func HandleMagicLinkPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		data := pages.MagicLinkPageData{}

		if err := pages.MagicLink(data).Render(ctx, w); err != nil {
			slog.Error("failed to render magic link page", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleRequestMagicLink processes the magic link request form.
func HandleRequestMagicLink(queries *store.Queries, mailer *email.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "type", "request", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		emailAddr := strings.TrimSpace(r.FormValue("email"))

		// Validate email format
		if err := validate.Email(emailAddr); err != nil {
			renderMagicLinkError(w, ctx, err.Error())
			return
		}

		// Always show success message to prevent email enumeration
		// Even if the email doesn't exist, we show the same message
		defer func() {
			renderMagicLinkSuccess(w, ctx)
		}()

		// Look up user by email
		user, err := queries.GetUserByEmail(ctx, sql.NullString{String: emailAddr, Valid: true})
		if err != nil {
			// User not found - silently succeed (no email enumeration)
			slog.Info("magic link requested for unknown email", "type", "request", "email", emailAddr)
			return
		}

		// Rate limit: max magic links per user per hour
		count, err := queries.CountRecentMagicLinksByUserID(ctx, user.ID)
		if err != nil {
			slog.Error("failed to count recent magic links", "type", "request", "error", err)
			return
		}
		if count >= maxMagicLinksPerHour {
			slog.Warn("magic link rate limit exceeded", "type", "request", "user_id", user.ID)
			return
		}

		// Generate magic link token
		token, err := auth.GenerateToken()
		if err != nil {
			slog.Error("failed to generate magic link token", "type", "request", "error", err)
			return
		}

		// Calculate expiry
		expiry := time.Now().UTC().Add(time.Duration(magicLinkExpiryHours) * time.Hour)

		// Store magic link
		_, err = queries.CreateMagicLink(ctx, store.CreateMagicLinkParams{
			Token:     token,
			UserID:    user.ID,
			ExpiresAt: expiry.Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			slog.Error("failed to create magic link", "type", "request", "error", err)
			return
		}

		// Send magic link email
		if mailer.Enabled() {
			if err := mailer.SendMagicLink(emailAddr, token); err != nil {
				slog.Error("failed to send magic link email", "type", "request", "error", err, "email", emailAddr)
				// Delete the magic link since email failed
				_ = queries.DeleteMagicLink(ctx, token)
				return
			}
		} else {
			slog.Warn("email not configured, magic link created but email not sent", "type", "request", "token", token)
		}

		slog.Info("magic link sent", "type", "request", "user_id", user.ID)
	}
}

// HandleMagicLinkLogin processes magic link authentication.
func HandleMagicLinkLogin(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := r.PathValue("token")

		// Validate magic link token and get user
		row, err := queries.GetMagicLinkWithUser(ctx, token)
		if err != nil {
			slog.Warn("invalid or expired magic link token", "type", "request")
			http.Error(w, "Invalid or expired login link", http.StatusNotFound)
			return
		}

		// Delete the magic link (one-time use)
		if err := queries.DeleteMagicLink(ctx, token); err != nil {
			slog.Warn("failed to delete magic link", "type", "request", "error", err)
		}

		// Create session
		sessionToken, err := auth.CreateSession(ctx, queries, row.UserID)
		if err != nil {
			slog.Error("failed to create session", "type", "request", "error", err)
			http.Error(w, "Failed to log in", http.StatusInternalServerError)
			return
		}

		setSessionCookie(w, sessionToken)
		slog.Info("user logged in via magic link", "type", "request", "user_id", row.UserID, "username", row.Username)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// renderMagicLinkError renders the magic link page with an error message.
func renderMagicLinkError(w http.ResponseWriter, ctx context.Context, errMsg string) {
	data := pages.MagicLinkPageData{
		Error: errMsg,
	}

	w.WriteHeader(http.StatusBadRequest)
	pages.MagicLink(data).Render(ctx, w)
}

// renderMagicLinkSuccess renders the magic link page with a success message.
func renderMagicLinkSuccess(w http.ResponseWriter, ctx context.Context) {
	data := pages.MagicLinkPageData{
		Success: true,
	}

	pages.MagicLink(data).Render(ctx, w)
}
