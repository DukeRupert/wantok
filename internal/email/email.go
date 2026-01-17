// Package email provides email sending functionality via SMTP or Postmark.
package email

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Provider specifies the email service provider.
type Provider string

const (
	ProviderSMTP     Provider = "smtp"
	ProviderPostmark Provider = "postmark"
)

// Config holds email configuration.
type Config struct {
	Provider Provider

	// SMTP configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPTLS      bool

	// Postmark configuration
	PostmarkServerToken string

	// Common configuration
	From string
}

// Mailer handles email sending.
type Mailer struct {
	config     Config
	baseURL    string
	httpClient *http.Client
}

// New creates a new Mailer with the given configuration.
func New(cfg Config, baseURL string) *Mailer {
	return &Mailer{
		config:  cfg,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendInvitation sends an invitation email with a registration link.
func (m *Mailer) SendInvitation(to, token string) error {
	link := fmt.Sprintf("%s/register/%s", m.baseURL, token)

	subject := "You're invited to join Wantok"
	textBody := fmt.Sprintf(`Hello,

You've been invited to join Wantok, a family messaging app.

Click the link below to create your account:
%s

This link will expire in 7 days.

If you didn't expect this invitation, you can safely ignore this email.

- The Wantok Family`, link)

	htmlBody := fmt.Sprintf(`<p>Hello,</p>
<p>You've been invited to join Wantok, a family messaging app.</p>
<p><a href="%s">Click here to create your account</a></p>
<p>Or copy this link: %s</p>
<p>This link will expire in 7 days.</p>
<p>If you didn't expect this invitation, you can safely ignore this email.</p>
<p>- The Wantok Family</p>`, link, link)

	return m.send(to, subject, textBody, htmlBody)
}

// SendMagicLink sends a magic link email for passwordless login.
func (m *Mailer) SendMagicLink(to, token string) error {
	link := fmt.Sprintf("%s/auth/magic/%s", m.baseURL, token)

	subject := "Your Wantok login link"
	textBody := fmt.Sprintf(`Hello,

Click the link below to sign in to Wantok:
%s

This link will expire in 24 hours and can only be used once.

If you didn't request this link, you can safely ignore this email.

- The Wantok Family`, link)

	htmlBody := fmt.Sprintf(`<p>Hello,</p>
<p><a href="%s">Click here to sign in to Wantok</a></p>
<p>Or copy this link: %s</p>
<p>This link will expire in 24 hours and can only be used once.</p>
<p>If you didn't request this link, you can safely ignore this email.</p>
<p>- The Wantok Family</p>`, link, link)

	return m.send(to, subject, textBody, htmlBody)
}

// send sends an email using the configured provider.
func (m *Mailer) send(to, subject, textBody, htmlBody string) error {
	switch m.config.Provider {
	case ProviderPostmark:
		return m.sendPostmark(to, subject, textBody, htmlBody)
	case ProviderSMTP:
		return m.sendSMTP(to, subject, textBody)
	default:
		// Default to SMTP for backwards compatibility
		return m.sendSMTP(to, subject, textBody)
	}
}

// Enabled returns true if the mailer is configured and ready to send emails.
func (m *Mailer) Enabled() bool {
	switch m.config.Provider {
	case ProviderPostmark:
		return m.config.PostmarkServerToken != ""
	case ProviderSMTP:
		return m.config.SMTPHost != ""
	default:
		return m.config.SMTPHost != "" || m.config.PostmarkServerToken != ""
	}
}

// postmarkEmail represents the Postmark API email payload.
type postmarkEmail struct {
	From     string `json:"From"`
	To       string `json:"To"`
	Subject  string `json:"Subject"`
	TextBody string `json:"TextBody,omitempty"`
	HtmlBody string `json:"HtmlBody,omitempty"`
}

// postmarkResponse represents the Postmark API response.
type postmarkResponse struct {
	ErrorCode int    `json:"ErrorCode"`
	Message   string `json:"Message"`
	MessageID string `json:"MessageID"`
}

// sendPostmark sends an email via Postmark API.
func (m *Mailer) sendPostmark(to, subject, textBody, htmlBody string) error {
	email := postmarkEmail{
		From:     m.config.From,
		To:       to,
		Subject:  subject,
		TextBody: textBody,
		HtmlBody: htmlBody,
	}

	payload, err := json.Marshal(email)
	if err != nil {
		return fmt.Errorf("failed to marshal email: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.postmarkapp.com/email", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Postmark-Server-Token", m.config.PostmarkServerToken)

	resp, err := m.httpClient.Do(req)
	if err != nil {
		slog.Error("failed to send email via Postmark", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var pmResp postmarkResponse
		if err := json.Unmarshal(body, &pmResp); err == nil && pmResp.Message != "" {
			slog.Error("Postmark API error", "to", to, "status", resp.StatusCode, "error", pmResp.Message, "code", pmResp.ErrorCode)
			return fmt.Errorf("Postmark error: %s (code %d)", pmResp.Message, pmResp.ErrorCode)
		}
		slog.Error("Postmark API error", "to", to, "status", resp.StatusCode, "body", string(body))
		return fmt.Errorf("Postmark error: status %d", resp.StatusCode)
	}

	var pmResp postmarkResponse
	if err := json.Unmarshal(body, &pmResp); err == nil {
		slog.Info("email sent via Postmark", "to", to, "subject", subject, "message_id", pmResp.MessageID)
	} else {
		slog.Info("email sent via Postmark", "to", to, "subject", subject)
	}

	return nil
}

// sendSMTP sends an email via SMTP.
func (m *Mailer) sendSMTP(to, subject, body string) error {
	// Build the email message
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", m.config.From, to, subject, body)

	addr := fmt.Sprintf("%s:%d", m.config.SMTPHost, m.config.SMTPPort)

	var auth smtp.Auth
	if m.config.SMTPUsername != "" {
		auth = smtp.PlainAuth("", m.config.SMTPUsername, m.config.SMTPPassword, m.config.SMTPHost)
	}

	if m.config.SMTPTLS {
		return m.sendSMTPTLS(addr, auth, to, []byte(msg))
	}

	err := smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(msg))
	if err != nil {
		slog.Error("failed to send email via SMTP", "to", to, "error", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	slog.Info("email sent via SMTP", "to", to, "subject", subject)
	return nil
}

// sendSMTPTLS sends an email using explicit TLS connection.
func (m *Mailer) sendSMTPTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	// Connect to the server
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: m.config.SMTPHost,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.config.SMTPHost)
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

	slog.Info("email sent via SMTP TLS", "to", to)
	return nil
}
