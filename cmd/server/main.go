package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dukerupert/wantok/internal/auth"
	"github.com/dukerupert/wantok/internal/database"
	"github.com/dukerupert/wantok/internal/handlers"
	"github.com/dukerupert/wantok/internal/render"
	"github.com/dukerupert/wantok/internal/store"
	"golang.org/x/term"
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

// createAdmin prompts for credentials and creates an admin user.
// Exits after completion (does not start server).
func createAdmin(cfg AppConfig) error {
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return err
	}
	slog.Info("database connection established")
	queries := store.New(db)

	username, err := promptString("username: ")
	if err != nil {
		return fmt.Errorf("no username input received: %w", err)
	}

	password, err := promptPassword("password: ")
	if err != nil {
		return fmt.Errorf("no password input received: %w", err)
	}

	displayName, err := promptString("display name: ")
	if err != nil {
		return fmt.Errorf("no display name received: %w", err)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	ctx := context.Background()
	_, err = queries.CreateUser(ctx, store.CreateUserParams{
		Username: username,
		DisplayName: displayName,
		PasswordHash: hash,
		IsAdmin: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	fmt.Printf("Admin user '%s' created successfully\n", username)
	return nil
}

// promptString reads a line of input from stdin with the given prompt.
func promptString(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password from stdin without echoing.
func promptPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // newline after hidden input
	return string(bytes), nil
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
	// Parse command-line flags
	createAdminFlag := flag.Bool("create-admin", false, "Create an admin user and exit")
	flag.Parse()

	cfg := loadConfig(os.Environ())

	// Handle --create-admin flag
	if *createAdminFlag {
		if err := createAdmin(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating admin: %s\n", err)
			os.Exit(1)
		}
		return
	}

	// Run the server
	ctx := context.Background()
	if err := run(ctx, os.Stdout, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
