package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strconv"

	"github.com/dukerupert/wantok/internal/database"
	_ "modernc.org/sqlite" // Import the modernc.org/sqlite driver
)

type AppConfig struct {
	DatabasePath  string
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
		ListenAddr:    ":8080",
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
	_, err := database.New(cfg.DatabasePath)
	if err != nil {
		return err
	}
	slog.Info("database connection established")
	err = errors.New("All done here")
	return err
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
