package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/dukerupert/wantok/internal/store"
)

// Cleaner handles periodic cleanup of expired data.
type Cleaner struct {
	queries  *store.Queries
	interval time.Duration
	stop     chan struct{}
}

// New creates a new Cleaner with the specified interval.
func New(queries *store.Queries, interval time.Duration) *Cleaner {
	return &Cleaner{
		queries:  queries,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Start begins the cleanup loop in a goroutine.
func (c *Cleaner) Start() {
	go c.run()
}

// Stop signals the cleanup loop to stop.
func (c *Cleaner) Stop() {
	close(c.stop)
}

func (c *Cleaner) run() {
	slog.Info("cleanup service started", "type", "lifecycle", "interval", c.interval.String())

	// Run immediately on start
	c.runCleanup()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.runCleanup()
		case <-c.stop:
			slog.Info("cleanup service stopped", "type", "lifecycle")
			return
		}
	}
}

func (c *Cleaner) runCleanup() {
	ctx := context.Background()

	// Delete expired sessions
	sessResult, err := c.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		slog.Error("failed to delete expired sessions", "type", "cleanup", "error", err)
	} else {
		if count, _ := sessResult.RowsAffected(); count > 0 {
			slog.Info("deleted expired sessions", "type", "cleanup", "count", count)
		}
	}

	// Delete old messages (30+ days)
	msgResult, err := c.queries.DeleteOldMessages(ctx)
	if err != nil {
		slog.Error("failed to delete old messages", "type", "cleanup", "error", err)
	} else {
		if count, _ := msgResult.RowsAffected(); count > 0 {
			slog.Info("deleted old messages", "type", "cleanup", "count", count)
		}
	}

	// Delete expired invitations
	invResult, err := c.queries.DeleteExpiredInvitations(ctx)
	if err != nil {
		slog.Error("failed to delete expired invitations", "type", "cleanup", "error", err)
	} else {
		if count, _ := invResult.RowsAffected(); count > 0 {
			slog.Info("deleted expired invitations", "type", "cleanup", "count", count)
		}
	}

	// Delete expired magic links
	mlResult, err := c.queries.DeleteExpiredMagicLinks(ctx)
	if err != nil {
		slog.Error("failed to delete expired magic links", "type", "cleanup", "error", err)
	} else {
		if count, _ := mlResult.RowsAffected(); count > 0 {
			slog.Info("deleted expired magic links", "type", "cleanup", "count", count)
		}
	}
}
