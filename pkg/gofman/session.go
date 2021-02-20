package gofman

import (
	"context"
)

// Session constants.
const (
	MinTokenLen = 32
)

// Session represents an active user session. These are linked to a user.
type Session struct {
	ID        string `json:"id"`
	UserID    string `json:"users_id"`
	Token     string `json:"token"`
	CreatedAt int64  `json:"created_at"`
}

// Validate returns an error if any fields are invalid in the session.
func (s *Session) Validate() error {
	if s.UserID == "" {
		return NewError(EINVALID, "User ID required.")
	}

	if s.Token == "" {
		return NewError(EINVALID, "Access token required.")
	}

	if len(s.Token) < MinTokenLen {
		return NewError(EINVALID, "Token must have at least %d characters.", MinTokenLen)
	}

	return nil
}

// CanDeleteSession returns true if the current user can remove the session.
func CanDeleteSession(ctx context.Context, session *Session) bool {
	if id := UserIDFromContext(ctx); id != "" && session.UserID == id {
		return true
	}

	return false
}

// SessionService represents a service for managing sessions. The functions
// should return ENOTFOUND if the session could not be found and EUNAUTHORIZED
// if the user is not authorized to run the transaction.
type SessionService interface {
	FindSessionForToken(ctx context.Context, id string, token string) (*Session, error)
	FindSessions(ctx context.Context, filter SessionFilter) ([]*Session, int, error)
	CreateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, id string) error
}

// SessionFilter represents a filter accepted by FindSessions().
type SessionFilter struct {
	ID     *string `json:"string"`
	UserID *string `json:"users_id"`
	Token  *string `json:"token"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}
