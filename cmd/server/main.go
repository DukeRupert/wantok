package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/dukerupert/wantok/internal/database"
	"github.com/dukerupert/wantok/internal/handlers"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
	_ "modernc.org/sqlite"
)

type AppConfig struct {
	DatabasePath  string
	Host          string
	ListenAddr    string
	SessionSecret string
	SessionMaxAge int
}

func getenv(target string, list []string) string {
	prefix := target + "="
	for _, item := range list {
		if len(item) > len(prefix) && item[:len(prefix)] == prefix {
			return item[len(prefix):]
		}
	}
	return ""
}

func loadConfig(args []string) AppConfig {
	// defaults
	cfg := AppConfig{
		DatabasePath:  "wantok.db",
		Host:          "localhost",
		ListenAddr:    "8080",
		SessionSecret: "PaxRomana",
		SessionMaxAge: 3600,
	}

	path := getenv("DATABASE_PATH", args)
	if path != "" {
		cfg.DatabasePath = path
	}

	port := getenv("PORT", args)
	if port != "" {
		cfg.ListenAddr = port
	}

	secret := getenv("SESSION_SECRET", args)
	if secret != "" {
		cfg.SessionSecret = secret
	}

	maxAge := getenv("SESSION_MAX_AGE", args)
	if maxAge != "" {
		i, err := strconv.Atoi(maxAge)
		if err != nil {
			slog.Info("Invalid session max age", "error", err)
			i = 3600
		}
		cfg.SessionMaxAge = i
	}
	return cfg
}

func run(ctx context.Context, w io.Writer, args []string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	cfg := loadConfig(args)
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return err
	}
	slog.Info("database connection established")
	queries := store.New(db)

	renderer, err := render.New()
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}
	slog.Info("template renderer initialized")

	srv := handlers.NewServer(queries, renderer)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.ListenAddr),
		Handler: srv,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutting down server")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
