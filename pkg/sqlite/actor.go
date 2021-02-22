package sqlite

import (
	"context"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.ActorService = (*ActorService)(nil)

// ActorService represents a service for managing actors.
type ActorService struct {
	db *DB
}

// NewActorService returns a new instance of ActorService.
func NewActorService(db *DB) *ActorService {
	return &ActorService{db: db}
}

// FindActorByID retrieves a actor by ID.
// Returns ENOTFOUND if actor does not exist.
func (s *ActorService) FindActorByID(ctx context.Context, id string) (*gofman.Actor, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	actor, err := findActorByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	return actor, nil
}

// FindActors retrieves actor objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func (s *ActorService) FindActors(ctx context.Context, filter gofman.ActorFilter) ([]*gofman.Actor, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback()

	actors, total, err := findActors(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	return actors, total, nil
}

// CreateActor creates a new actor.
func (s *ActorService) CreateActor(ctx context.Context, actor *gofman.Actor) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createActor(ctx, tx, actor); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateActor updates a actor object.
// Returns EUNAUTHORIZED if current user is not the creator of the actor.
// Returns ENOTFOUND if actor does not exist.
func (s *ActorService) UpdateActor(ctx context.Context, id string, update gofman.ActorUpdate) (*gofman.Actor, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	actor, err := updateActor(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return actor, nil
}

// RemoveActor sets the removed timestamp to the current time. This allows
// us to re-enable removed actor.
// Returns EUNAUTHORIZED if current user is not the creator of the actor.
// Returns ENOTFOUND if actor does not exist.
func (s *ActorService) RemoveActor(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := removeActor(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// findActorByID is a helper function to fetch a actor by ID.
// Returns ENOTFOUND if actor does not exist.
func findActorByID(ctx context.Context, tx *Tx, id string) (*gofman.Actor, error) {
	actors, _, err := findActors(ctx, tx, gofman.ActorFilter{ID: &id, Limit: 1})
	if err != nil {
		return nil, err
	}

	if len(actors) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "Actor not found.")
	}

	return actors[0], nil
}

// FindActors retrieves actor objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func findActors(ctx context.Context, tx *Tx, filter gofman.ActorFilter) ([]*gofman.Actor, int, error) {
	if gofman.CanFindActor(ctx, filter) == false {
		return nil, 0, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to search using this filter.")
	}

	where, args := []string{"1 = 1"}, []interface{}{}

	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	if v := filter.UserID; v != nil {
		where, args = append(where, "users_id = ?"), append(args, *v)
	}

	where = append(where, "removed_at = 0")

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			users_id,
			name,
			created_at,
			updated_at,
			removed_at,
			COUNT(*) OVER()
		FROM actors
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
	var actors []*gofman.Actor

	for rows.Next() {
		var actor gofman.Actor

		if err = rows.Scan(
			&actor.ID, &actor.UserID, &actor.Name,
			&actor.CreatedAt, &actor.UpdatedAt, &actor.RemovedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}

		actors = append(actors, &actor)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return actors, n, nil
}

// createActor creates a new actor.
func createActor(ctx context.Context, tx *Tx, actor *gofman.Actor) error {
	if err := actor.Validate(); err != nil {
		return err
	}

	if gofman.CanUpdateActor(ctx, actor) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to create this actor.")
	}

	if id, err := tx.db.ID(); err != nil {
		return err
	} else {
		actor.ID = id
	}

	actor.CreatedAt = tx.now
	actor.UpdatedAt = actor.CreatedAt

	_, err := tx.ExecContext(ctx, `
		INSERT INTO actors (
			id,
			users_id,
			name,
			created_at,
			updated_at,
			removed_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		actor.ID,
		actor.UserID,
		actor.Name,
		actor.CreatedAt,
		actor.UpdatedAt,
		0,
	)

	if err != nil {
		return err
	}

	return nil
}

// updateActor updates a actor object.
// Returns EUNAUTHORIZED if current user is not the creator of the actor.
// Returns ENOTFOUND if actor does not exist.
func updateActor(ctx context.Context, tx *Tx, id string, update gofman.ActorUpdate) (*gofman.Actor, error) {
	actor, err := findActorByID(ctx, tx, id)
	if err != nil {
		return actor, err
	}

	if gofman.CanUpdateActor(ctx, actor) == false {
		return nil, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to update this actor.")
	}

	if v := update.Name; v != nil {
		actor.Name = *v
	}

	actor.UpdatedAt = tx.now

	if err := actor.Validate(); err != nil {
		return actor, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE actors
		SET name = ?,
			updated_at = ?
		WHERE id = ?
	`,
		actor.Name,
		actor.UpdatedAt,
		id,
	)

	if err != nil {
		return actor, err
	}

	return actor, nil
}

// removeActor sets the removed timestamp to the current time. This allows
// us to re-enable removed actor.
// Returns EUNAUTHORIZED if current user is not the creator of the actor.
// Returns ENOTFOUND if actor does not exist.
func removeActor(ctx context.Context, tx *Tx, id string) error {
	actor, err := findActorByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if gofman.CanUpdateActor(ctx, actor) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to remove this actor.")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE actors
		SET removed_at = ?
		WHERE id = ?
	`,
		tx.now,
		id,
	)

	if err != nil {
		return err
	}

	return nil
}
