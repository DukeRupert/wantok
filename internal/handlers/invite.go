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

// timeFormat is the SQLite datetime format
const timeFormat = "2006-01-02 15:04:05"

const (
	invitationExpiryDays = 7
	maxInvitesPerHour    = 3
)

// HandleInviteUser processes the invite user form (admin only).
func HandleInviteUser(queries *store.Queries, mailer *email.Mailer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := auth.GetUser(ctx)

		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "type", "request", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		emailAddr := strings.TrimSpace(r.FormValue("email"))

		// Validate email
		if err := validate.Email(emailAddr); err != nil {
			renderAdminError(w, queries, ctx, user.ID, err.Error())
			return
		}

		// Check if email is already registered
		_, err := queries.GetUserByEmail(ctx, sql.NullString{String: emailAddr, Valid: true})
		if err == nil {
			renderAdminError(w, queries, ctx, user.ID, "A user with this email already exists")
			return
		}

		// Check for existing pending invitation
		_, err = queries.GetInvitationByEmail(ctx, emailAddr)
		if err == nil {
			renderAdminError(w, queries, ctx, user.ID, "An invitation for this email is already pending")
			return
		}

		// Rate limit: max invitations per email per hour
		count, err := queries.CountRecentInvitationsByEmail(ctx, emailAddr)
		if err != nil {
			slog.Error("failed to count recent invitations", "type", "request", "error", err)
			renderAdminError(w, queries, ctx, user.ID, "Failed to send invitation")
			return
		}
		if count >= maxInvitesPerHour {
			renderAdminError(w, queries, ctx, user.ID, "Too many invitations for this email. Please wait before trying again.")
			return
		}

		// Generate invitation token
		token, err := auth.GenerateToken()
		if err != nil {
			slog.Error("failed to generate invitation token", "type", "request", "error", err)
			renderAdminError(w, queries, ctx, user.ID, "Failed to send invitation")
			return
		}

		// Calculate expiry
		expiry := time.Now().UTC().AddDate(0, 0, invitationExpiryDays)

		// Store invitation
		_, err = queries.CreateInvitation(ctx, store.CreateInvitationParams{
			Token:     token,
			Email:     emailAddr,
			InvitedBy: user.ID,
			ExpiresAt: expiry.Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			slog.Error("failed to create invitation", "type", "request", "error", err)
			renderAdminError(w, queries, ctx, user.ID, "Failed to send invitation")
			return
		}

		// Send invitation email
		if mailer.Enabled() {
			if err := mailer.SendInvitation(emailAddr, token); err != nil {
				slog.Error("failed to send invitation email", "type", "request", "error", err, "email", emailAddr)
				// Delete the invitation since email failed
				_ = queries.DeleteInvitation(ctx, token)
				renderAdminError(w, queries, ctx, user.ID, "Failed to send invitation email")
				return
			}
		} else {
			slog.Warn("email not configured, invitation created but email not sent", "type", "request", "token", token)
		}

		slog.Info("invitation sent", "type", "request", "email", emailAddr, "invited_by", user.Username)
		http.Redirect(w, r, "/admin?invited="+emailAddr, http.StatusSeeOther)
	}
}

// HandleRegisterPage renders the registration form for invited users.
func HandleRegisterPage(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := r.PathValue("token")

		// Validate invitation token
		invitation, err := queries.GetInvitationByToken(ctx, token)
		if err != nil {
			slog.Warn("invalid or expired invitation token", "type", "request", "token", token)
			http.Error(w, "Invalid or expired invitation link", http.StatusNotFound)
			return
		}

		data := pages.RegisterPageData{
			Email: invitation.Email,
			Token: token,
		}

		if err := pages.Register(data).Render(ctx, w); err != nil {
			slog.Error("failed to render register page", "type", "request", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// HandleRegister processes the registration form for invited users.
func HandleRegister(queries *store.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		token := r.PathValue("token")

		// Validate invitation token
		invitation, err := queries.GetInvitationByToken(ctx, token)
		if err != nil {
			slog.Warn("invalid or expired invitation token", "type", "request", "token", token)
			http.Error(w, "Invalid or expired invitation link", http.StatusNotFound)
			return
		}

		if err := r.ParseForm(); err != nil {
			slog.Error("failed to parse form", "type", "request", "error", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		username := strings.TrimSpace(r.FormValue("username"))
		displayName := strings.TrimSpace(r.FormValue("display_name"))
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate input
		if err := validate.Username(username); err != nil {
			renderRegisterError(w, ctx, invitation.Email, token, err.Error())
			return
		}
		if err := validate.DisplayName(displayName); err != nil {
			renderRegisterError(w, ctx, invitation.Email, token, err.Error())
			return
		}
		if err := validate.Password(password); err != nil {
			renderRegisterError(w, ctx, invitation.Email, token, err.Error())
			return
		}
		if password != confirmPassword {
			renderRegisterError(w, ctx, invitation.Email, token, "Passwords do not match")
			return
		}

		// Hash password
		hash, err := auth.HashPassword(password)
		if err != nil {
			slog.Error("failed to hash password", "type", "request", "error", err)
			renderRegisterError(w, ctx, invitation.Email, token, "Failed to create account")
			return
		}

		// Create user with email
		newUser, err := queries.CreateUserWithEmail(ctx, store.CreateUserWithEmailParams{
			Username:     username,
			DisplayName:  displayName,
			PasswordHash: hash,
			Email:        sql.NullString{String: invitation.Email, Valid: true},
			IsAdmin:      0,
		})
		if err != nil {
			slog.Error("failed to create user", "type", "request", "error", err)
			renderRegisterError(w, ctx, invitation.Email, token, "Failed to create account (username may already exist)")
			return
		}

		// Delete the invitation (one-time use)
		if err := queries.DeleteInvitation(ctx, token); err != nil {
			slog.Warn("failed to delete invitation", "type", "request", "error", err)
		}

		// Create session and log user in
		sessionToken, err := auth.CreateSession(ctx, queries, newUser.ID)
		if err != nil {
			slog.Error("failed to create session", "type", "request", "error", err)
			// User was created, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		setSessionCookie(w, sessionToken)
		slog.Info("user registered via invitation", "type", "request", "username", username, "email", invitation.Email)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// renderRegisterError renders the registration page with an error message.
func renderRegisterError(w http.ResponseWriter, ctx context.Context, emailAddr, token, errMsg string) {
	data := pages.RegisterPageData{
		Email: emailAddr,
		Token: token,
		Error: errMsg,
	}

	w.WriteHeader(http.StatusBadRequest)
	pages.Register(data).Render(ctx, w)
}
