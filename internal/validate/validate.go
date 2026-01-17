package validate

import (
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	// usernameRegex allows alphanumeric characters and underscores
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	// emailRegex is a basic email validation pattern
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	ErrUsernameEmpty      = errors.New("username is required")
	ErrUsernameTooShort   = errors.New("username must be at least 3 characters")
	ErrUsernameTooLong    = errors.New("username must be at most 32 characters")
	ErrUsernameInvalid    = errors.New("username must contain only letters, numbers, and underscores")
	ErrDisplayNameEmpty   = errors.New("display name is required")
	ErrDisplayNameTooLong = errors.New("display name must be at most 64 characters")
	ErrPasswordEmpty      = errors.New("password is required")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must be at most 128 characters")
	ErrMessageEmpty       = errors.New("message cannot be empty")
	ErrMessageTooLong     = errors.New("message must be at most 4096 characters")
	ErrEmailEmpty         = errors.New("email is required")
	ErrEmailTooLong       = errors.New("email must be at most 254 characters")
	ErrEmailInvalid       = errors.New("invalid email address")
)

// Username validates a username.
// Must be 3-32 characters, alphanumeric with underscores.
func Username(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return ErrUsernameEmpty
	}
	if utf8.RuneCountInString(s) < 3 {
		return ErrUsernameTooShort
	}
	if utf8.RuneCountInString(s) > 32 {
		return ErrUsernameTooLong
	}
	if !usernameRegex.MatchString(s) {
		return ErrUsernameInvalid
	}
	return nil
}

// DisplayName validates a display name.
// Must be 1-64 characters.
func DisplayName(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return ErrDisplayNameEmpty
	}
	if utf8.RuneCountInString(s) > 64 {
		return ErrDisplayNameTooLong
	}
	return nil
}

// Password validates a password.
// Must be 8-128 characters.
func Password(s string) error {
	if s == "" {
		return ErrPasswordEmpty
	}
	if utf8.RuneCountInString(s) < 8 {
		return ErrPasswordTooShort
	}
	if utf8.RuneCountInString(s) > 128 {
		return ErrPasswordTooLong
	}
	return nil
}

// Message validates a message content.
// Must be 1-4096 characters after trimming.
func Message(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return ErrMessageEmpty
	}
	if utf8.RuneCountInString(s) > 4096 {
		return ErrMessageTooLong
	}
	return nil
}

// Email validates an email address.
// Must be a valid email format and at most 254 characters.
func Email(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return ErrEmailEmpty
	}
	if len(s) > 254 {
		return ErrEmailTooLong
	}
	if !emailRegex.MatchString(s) {
		return ErrEmailInvalid
	}
	return nil
}
