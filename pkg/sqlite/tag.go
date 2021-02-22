package sqlite

import (
	"context"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.TagService = (*TagService)(nil)

// TagService represents a service for managing tags.
type TagService struct {
	db *DB
}

// NewTagService returns a new instance of TagService.
func NewTagService(db *DB) *TagService {
	return &TagService{db: db}
}

// FindTagByID retrieves a tag by ID.
// Returns ENOTFOUND if tag does not exist.
func (s *TagService) FindTagByID(ctx context.Context, id string) (*gofman.Tag, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	tag, err := findTagByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	return tag, nil
}

// FindTags retrieves tag objects and total hits based on a filter. The total
// hits may differ from the length of the slice if a limit was applied.
func (s *TagService) FindTags(ctx context.Context, filter gofman.TagFilter) ([]*gofman.Tag, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback()

	tags, total, err := findTags(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	return tags, total, nil
}

// CreateTag creates a new tag.
func (s *TagService) CreateTag(ctx context.Context, tag *gofman.Tag) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createTag(ctx, tx, tag); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateTag updates a tag object.
// Returns EUNAUTHORIZED if current user is not the creator of the tag.
// Returns ENOTFOUND if tag does not exist.
func (s *TagService) UpdateTag(ctx context.Context, id string, update gofman.TagUpdate) (*gofman.Tag, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	tag, err := updateTag(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return tag, nil
}

// RemoveTag sets the removed timestamp to the current time. This allows us
// to re-enable removed tag.
// Returns EUNAUTHORIZED if current user is not the creator of the tag.
// Returns ENOTFOUND if tag does not exist.
func (s *TagService) RemoveTag(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := removeTag(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// findTagByID retrieves a tag by ID.
// Returns ENOTFOUND if tag does not exist.
func findTagByID(ctx context.Context, tx *Tx, id string) (*gofman.Tag, error) {
	tags, _, err := findTags(ctx, tx, gofman.TagFilter{ID: &id, Limit: 1})
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "Tag not found.")
	}

	return tags[0], nil
}

// findTags retrieves tag objects and total hits based on a filter. The total
// hits may differ from the length of the slice if a limit was applied.
func findTags(ctx context.Context, tx *Tx, filter gofman.TagFilter) ([]*gofman.Tag, int, error) {
	if gofman.CanFindTag(ctx, filter) == false {
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
		FROM tags
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
	var tags []*gofman.Tag

	for rows.Next() {
		var tag gofman.Tag

		if err = rows.Scan(
			&tag.ID, &tag.UserID, &tag.Name,
			&tag.CreatedAt, &tag.UpdatedAt, &tag.RemovedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}

		tags = append(tags, &tag)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return tags, n, nil
}

// createTag creates a new tag.
func createTag(ctx context.Context, tx *Tx, tag *gofman.Tag) error {
	if err := tag.Validate(); err != nil {
		return err
	}

	if gofman.CanUpdateTag(ctx, tag) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to create this tag.")
	}

	if id, err := tx.db.ID(); err != nil {
		return err
	} else {
		tag.ID = id
	}

	tag.CreatedAt = tx.now
	tag.UpdatedAt = tag.CreatedAt

	_, err := tx.ExecContext(ctx, `
		INSERT INTO tags (
			id,
			users_id,
			name,
			created_at,
			updated_at,
			removed_at
		)
		VALUES (?, ?, ?, ?, ?, ?)
	`,
		tag.ID,
		tag.UserID,
		tag.Name,
		tag.CreatedAt,
		tag.UpdatedAt,
		0,
	)

	if err != nil {
		return err
	}

	return nil
}

// updateTag updates a tag object.
// Returns EUNAUTHORIZED if current user is not the creator of the tag.
// Returns ENOTFOUND if tag does not exist.
func updateTag(ctx context.Context, tx *Tx, id string, update gofman.TagUpdate) (*gofman.Tag, error) {
	tag, err := findTagByID(ctx, tx, id)
	if err != nil {
		return tag, err
	}

	if gofman.CanUpdateTag(ctx, tag) == false {
		return nil, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to update this tag.")
	}

	if v := update.Name; v != nil {
		tag.Name = *v
	}

	if err := tag.Validate(); err != nil {
		return tag, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tags
		SET name = ?,
		WHERE id = ?
	`,
		tag.Name,
		id,
	)

	if err != nil {
		return tag, err
	}

	return tag, nil
}

// removeTag sets the removed timestamp to the current time. This allows us
// to re-enable removed tag.
// Returns EUNAUTHORIZED if current user is not the creator of the tag.
// Returns ENOTFOUND if tag does not exist.
func removeTag(ctx context.Context, tx *Tx, id string) error {
	tag, err := findTagByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if gofman.CanUpdateTag(ctx, tag) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to remove this tag.")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE tags
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
