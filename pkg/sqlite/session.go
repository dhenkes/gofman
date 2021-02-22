package sqlite

import (
	"context"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.SessionService = (*SessionService)(nil)

// SessionService represents a service for managing sessions.
type SessionService struct {
	db *DB
}

// NewSessionService returns a new instance of SessionService.
func NewSessionService(db *DB) *SessionService {
	return &SessionService{db: db}
}

// FindSessionForToken looks up a session by ID and token.
// Returns ENOTFOUND if session does not exist.
func (s *SessionService) FindSessionForToken(ctx context.Context, id string, token string) (*gofman.Session, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	session, err := findSessionForToken(ctx, tx, id, token)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// FindSessions retrieves session objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func (s *SessionService) FindSessions(ctx context.Context, filter gofman.SessionFilter) ([]*gofman.Session, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback()

	sessions, total, err := findSessions(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// CreateSession creates a new session object.
func (s *SessionService) CreateSession(ctx context.Context, session *gofman.Session) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err = createSession(ctx, tx, session); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteSession permanently deletes a session object from the system by ID.
// Returns EUNAUTHORIZED if current user is not the creator of the session.
// Returns ENOTFOUND if session does not exist.
func (s *SessionService) DeleteSession(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err = deleteSession(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// findSessionByID looks up a session by ID.
// Returns ENOTFOUND if session does not exist.
func findSessionByID(ctx context.Context, tx *Tx, id string) (*gofman.Session, error) {
	sessions, _, err := findSessions(ctx, tx, gofman.SessionFilter{ID: &id, Limit: 1})

	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "Session not found.")
	}

	return sessions[0], nil
}

// findSessionForToken looks up a session by ID, user ID and token.
// Returns ENOTFOUND if session does not exist.
func findSessionForToken(ctx context.Context, tx *Tx, id string, token string) (*gofman.Session, error) {
	sessions, _, err := findSessions(ctx, tx, gofman.SessionFilter{ID: &id, Token: &token, Limit: 1})

	if err != nil {
		return nil, err
	}

	if len(sessions) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "Session not found.")
	}

	return sessions[0], nil
}

// findSessions retrieves session objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func findSessions(ctx context.Context, tx *Tx, filter gofman.SessionFilter) ([]*gofman.Session, int, error) {
	where, args := []string{"1 = 1"}, []interface{}{}

	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	if v := filter.UserID; v != nil {
		where, args = append(where, "users_id = ?"), append(args, *v)
	}

	if v := filter.Token; v != nil {
		where, args = append(where, "token = ?"), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			users_id,
			token,
			created_at,
			COUNT(*) OVER()
		FROM sessions
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY created_at ASC
		`+formatLimitOffset(filter.Limit, filter.Offset),
		args...,
	)

	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var n int
	var sessions []*gofman.Session

	for rows.Next() {
		var session gofman.Session

		if err = rows.Scan(
			&session.ID, &session.UserID, &session.Token,
			&session.CreatedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}

		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return sessions, n, nil
}

// createSession creates a new session object.
func createSession(ctx context.Context, tx *Tx, session *gofman.Session) error {
	if err := session.Validate(); err != nil {
		return err
	}

	if id, err := tx.db.ID(); err != nil {
		return err
	} else {
		session.ID = id
	}

	session.CreatedAt = tx.now

	_, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (
			id,
			users_id,
			token,
			created_at
		)
		VALUES (?, ?, ?, ?)
	`,
		session.ID,
		session.UserID,
		session.Token,
		session.CreatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// deleteSession permanently deletes a session object from the system by ID.
// Returns EUNAUTHORIZED if current user is not the creator of the session.
// Returns ENOTFOUND if session does not exist.
func deleteSession(ctx context.Context, tx *Tx, id string) error {
	session, err := findSessionByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if gofman.CanDeleteSession(ctx, session) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to delete this session.")
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id); err != nil {
		return err
	}

	return nil
}
