// Package email provides SMTP email sending functionality.
package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// Config holds SMTP configuration.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

// Mailer handles email sending via SMTP.
type Mailer struct {
	config  Config
	baseURL string
}

// New creates a new Mailer with the given configuration.
func New(cfg Config, baseURL string) *Mailer {
	return &Mailer{
		config:  cfg,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

// SendInvitation sends an invitation email with a registration link.
func (m *Mailer) SendInvitation(to, token string) error {
	link := fmt.Sprintf("%s/register/%s", m.baseURL, token)

	subject := "You're invited to join Wantok"
	body := fmt.Sprintf(`Hello,

You've been invited to join Wantok, a family messaging app.

Click the link below to create your account:
%s

This link will expire in 7 days.

If you didn't expect this invitation, you can safely ignore this email.

- The Wantok Family`, link)

	return m.send(to, subject, body)
}

// SendMagicLink sends a magic link email for passwordless login.
func (m *Mailer) SendMagicLink(to, token string) error {
	link := fmt.Sprintf("%s/auth/magic/%s", m.baseURL, token)

	subject := "Your Wantok login link"
	body := fmt.Sprintf(`Hello,

Click the link below to sign in to Wantok:
%s

This link will expire in 24 hours and can only be used once.

If you didn't request this link, you can safely ignore this email.

- The Wantok Family`, link)

	return m.send(to, subject, body)
}

// send sends an email via SMTP.
func (m *Mailer) send(to, subject, body string) error {
	// Build the email message
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", m.config.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	var auth smtp.Auth
	if m.config.Username != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	if m.config.TLS {
		return m.sendTLS(addr, auth, to, []byte(msg))
	}

	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(msg))
	if err != nil {
		slog.Error("failed to send email", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("email sent", "to", to, "subject", subject)
	return nil
}

// sendTLS sends an email using explicit TLS connection.
func (m *Mailer) sendTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	// Connect to the server
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: m.config.Host,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP auth failed: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(m.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send the message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to open data writer: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		slog.Warn("SMTP quit failed", "error", err)
	}

	slog.Info("email sent via TLS", "to", to)
	return nil
}

// Enabled returns true if the mailer is configured and ready to send emails.
func (m *Mailer) Enabled() bool {
	return m.config.Host != ""
}
