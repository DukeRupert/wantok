package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dukerupert/wantok/internal/store"
)

const (
	tokenLength       = 32 // 32 bytes = 64 hex characters
	sessionMaxAgeDays = 30
)

// GenerateToken creates a cryptographically secure random token.
// Returns a 64-character hex string (32 random bytes).
func GenerateToken() (string, error) {
	// Generate tokenLength random bytes using crypto/rand
	b := make([]byte, tokenLength)
	// Hex encode and return
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession generates a new session token and stores it in the database.
// Returns the token string for setting in a cookie.
func CreateSession(ctx context.Context, queries *store.Queries, userID int64) (string, error) {
	// Generate token
	token, err := GenerateToken(); if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}
	// Calculate expiry (now + sessionMaxAgeDays)
	expiry := time.Now().UTC().AddDate(0, 0, sessionMaxAgeDays)
	params := store.CreateSessionParams{
		Token: token,
		UserID: userID,
		ExpiresAt: expiry.Format("2006-01-02 15:04:05"),
	}
	_, err = queries.CreateSession(ctx, params); if err != nil {
		return "", fmt.Errorf("failed to create session in store: %w", err)
	}

	return token, nil
}

// ValidateSession checks if a token is valid and returns the associated user data.
// Returns nil and an error if the session is invalid or expired.
func ValidateSession(ctx context.Context, queries *store.Queries, token string) (*store.GetSessionWithUserRow, error) {
	user, err := queries.GetSessionWithUser(ctx, token); if err != nil {
		return nil, fmt.Errorf("failed to retrieve session with user: %w", err)
	}
	// query handles expiry checking
	return &user, nil
}

// DeleteSession removes a session from the database.
func DeleteSession(ctx context.Context, queries *store.Queries, token string) error {
	err := queries.DeleteSession(ctx, token); if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions removes all sessions for a user.
// Use when a user changes password or is deleted.
func DeleteUserSessions(ctx context.Context, queries *store.Queries, userID int64) error {
	err := queries.DeleteUserSessions(ctx, userID); if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}