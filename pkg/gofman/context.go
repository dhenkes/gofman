package gofman

import "context"

// contextKey represents an internal key for adding context fields.
type contextKey int

// List of context keys.
const (
	requestIDContextKey = contextKey(iota + 1)
	userContextKey      = contextKey(iota + 1)
	sessionContextKey   = contextKey(iota + 1)
)

// NewContextWithRequestID returns a new context with the given request id.
func NewContextWithRequestID(ctx context.Context, v string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, v)
}

// RequestIDFromContext returns the current request id.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDContextKey).(string)
	return v
}

// NewContextWithUser returns a new context with the given user.
func NewContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext returns the current logged in user.
func UserFromContext(ctx context.Context) *User {
	v, _ := ctx.Value(userContextKey).(*User)
	return v
}

// UserIDFromContext is a helper function that returns the ID of the current
// logged in user. Returns an empty string if no user is logged in.
func UserIDFromContext(ctx context.Context) string {
	if user := UserFromContext(ctx); user != nil {
		return user.ID
	}

	return ""
}

// NewContextWithSession returns a new context with the current session.
func NewContextWithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// SessionFromContext returns the current session.
func SessionFromContext(ctx context.Context) *Session {
	v, _ := ctx.Value(sessionContextKey).(*Session)
	return v
}
