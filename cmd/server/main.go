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
	"github.com/dukerupert/wantok/internal/cleanup"
	"github.com/dukerupert/wantok/internal/database"
	"github.com/dukerupert/wantok/internal/email"
	"github.com/dukerupert/wantok/internal/handlers"
	"github.com/dukerupert/wantok/internal/realtime"
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
	SecureCookies bool

	// Email provider: "postmark" or "smtp"
	EmailProvider string

	// SMTP configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPTLS      bool

	// Postmark configuration
	PostmarkServerToken string

	// Common email configuration
	EmailFrom string

	// Base URL for email links
	BaseURL string
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
		SecureCookies: true, // Default to secure (production)
		SMTPPort:      587,
		SMTPTLS:       true,
	}

	path := getenv("DATABASE_PATH", args)
	if path != "" {
		cfg.DatabasePath = path
	}

	host := getenv("HOST", args)
	if host != "" {
		cfg.Host = host
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
			slog.Info("Invalid session max age", "type", "lifecycle", "error", err)
			i = 3600
		}
		cfg.SessionMaxAge = i
	}

	// SECURE_COOKIES=false disables Secure flag for local development
	if getenv("SECURE_COOKIES", args) == "false" {
		cfg.SecureCookies = false
	}

	// Email provider selection (defaults to "postmark" if POSTMARK_SERVER_TOKEN is set, else "smtp")
	cfg.EmailProvider = getenv("EMAIL_PROVIDER", args)
	cfg.PostmarkServerToken = getenv("POSTMARK_SERVER_TOKEN", args)

	// Auto-detect provider if not explicitly set
	if cfg.EmailProvider == "" {
		if cfg.PostmarkServerToken != "" {
			cfg.EmailProvider = "postmark"
		} else {
			cfg.EmailProvider = "smtp"
		}
	}

	// Common email config
	cfg.EmailFrom = getenv("EMAIL_FROM", args)

	// SMTP configuration
	cfg.SMTPHost = getenv("SMTP_HOST", args)
	cfg.SMTPUsername = getenv("SMTP_USERNAME", args)
	cfg.SMTPPassword = getenv("SMTP_PASSWORD", args)

	smtpPort := getenv("SMTP_PORT", args)
	if smtpPort != "" {
		if p, err := strconv.Atoi(smtpPort); err == nil {
			cfg.SMTPPort = p
		}
	}

	if getenv("SMTP_TLS", args) == "false" {
		cfg.SMTPTLS = false
	}

	cfg.BaseURL = getenv("BASE_URL", args)

	return cfg
}

// createAdmin prompts for credentials and creates an admin user.
// Exits after completion (does not start server).
func createAdmin(cfg AppConfig) error {
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		return err
	}
	slog.Info("database connection established", "type", "lifecycle")
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
	slog.Info("database connection established", "type", "lifecycle")
	queries := store.New(db)

	// Set secure cookies based on config
	handlers.SecureCookies = cfg.SecureCookies
	slog.Info("cookie security configured", "type", "lifecycle", "secure", cfg.SecureCookies)

	// Create email mailer
	mailer := email.New(email.Config{
		Provider:            email.Provider(cfg.EmailProvider),
		SMTPHost:            cfg.SMTPHost,
		SMTPPort:            cfg.SMTPPort,
		SMTPUsername:        cfg.SMTPUsername,
		SMTPPassword:        cfg.SMTPPassword,
		SMTPTLS:             cfg.SMTPTLS,
		PostmarkServerToken: cfg.PostmarkServerToken,
		From:                cfg.EmailFrom,
	}, cfg.BaseURL)
	if mailer.Enabled() {
		slog.Info("email service configured", "type", "lifecycle", "provider", cfg.EmailProvider)
	} else {
		slog.Warn("email service not configured - invitations and magic links will not work", "type", "lifecycle")
	}

	// Create and start WebSocket hub
	hub := realtime.NewHub()
	go hub.Run()

	// Start cleanup service (runs every hour)
	cleaner := cleanup.New(queries, time.Hour)
	cleaner.Start()
	defer cleaner.Stop()

	srv := handlers.NewServer(queries, hub, mailer)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.ListenAddr),
		Handler: srv,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("server started", "type", "lifecycle", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutting down server", "type", "lifecycle")
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
