package sqlite

import (
	"context"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.SetupService = (*SetupService)(nil)

// SetupService represents a service for checking if the setup should be
// executed.
type SetupService struct {
	db *DB
}

// NewSetupService returns a new instance of SetupService.
func NewSetupService(db *DB) *SetupService {
	return &SetupService{db: db}
}

// ShouldRunSetup checks if users exist. If that is not the case it will
// return true.
func (s *SetupService) ShouldRunSetup(ctx context.Context) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}

	defer tx.Rollback()

	users, _, err := findUsers(ctx, tx, gofman.UserFilter{Limit: 1})
	if err != nil {
		return false, err
	}

	return (len(users) > 0), nil
}
