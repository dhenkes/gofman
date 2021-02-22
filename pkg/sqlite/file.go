package sqlite

import (
	"context"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.FileService = (*FileService)(nil)

// FileService represents a service for managing files.
type FileService struct {
	db *DB
}

// NewFileService returns a new instance of FileService.
func NewFileService(db *DB) *FileService {
	return &FileService{db: db}
}

// FindFileByID retrieves a file by ID.
// Returns ENOTFOUND if file does not exist.
func (s *FileService) FindFileByID(ctx context.Context, id string) (*gofman.File, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	file, err := findFileByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// FindFiles retrieves file objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func (s *FileService) FindFiles(ctx context.Context, filter gofman.FileFilter) ([]*gofman.File, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback()

	files, total, err := findFiles(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	return files, total, nil
}

// CreateFile creates a new file.
func (s *FileService) CreateFile(ctx context.Context, file *gofman.File) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createFile(ctx, tx, file); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateFile updates a file object.
// Returns EUNAUTHORIZED if current user is not the creator of the file.
// Returns ENOTFOUND if file does not exist.
func (s *FileService) UpdateFile(ctx context.Context, id string, update gofman.FileUpdate) (*gofman.File, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	file, err := updateFile(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return file, nil
}

// RemoveFile sets the removed timestamp to the current time. This allows
// us to re-enable removed file.
// Returns EUNAUTHORIZED if current user is not the creator of the file.
// Returns ENOTFOUND if file does not exist.
func (s *FileService) RemoveFile(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := removeFile(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// findFileByID is a helper function to fetch a file by ID.
// Returns ENOTFOUND if file does not exist.
func findFileByID(ctx context.Context, tx *Tx, id string) (*gofman.File, error) {
	files, _, err := findFiles(ctx, tx, gofman.FileFilter{ID: &id, Limit: 1})
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "File not found.")
	}

	return files[0], nil
}

// FindFiles retrieves file objects and total hits based on a filter.
// The total hits may differ from the length of the slice if a limit was
// applied.
func findFiles(ctx context.Context, tx *Tx, filter gofman.FileFilter) ([]*gofman.File, int, error) {
	if gofman.CanFindFile(ctx, filter) == false {
		return nil, 0, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to search using this filter.")
	}

	where, args := []string{"1 = 1"}, []interface{}{}

	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	if v := filter.UserID; v != nil {
		where, args = append(where, "users_id = ?"), append(args, *v)
	}

	if v := filter.Type; v != nil {
		where, args = append(where, "type = ?"), append(args, *v)
	}

	where = append(where, "removed_at = 0")

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			users_id,
			name,
			type,
			path,
			checksum,
			created_at,
			updated_at,
			removed_at,
			COUNT(*) OVER()
		FROM files
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
	var files []*gofman.File

	for rows.Next() {
		var file gofman.File

		if err = rows.Scan(
			&file.ID, &file.UserID, &file.Name, &file.Type, &file.Path, &file.Checksum,
			&file.CreatedAt, &file.UpdatedAt, &file.RemovedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}

		files = append(files, &file)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return files, n, nil
}

// createFile creates a new file.
func createFile(ctx context.Context, tx *Tx, file *gofman.File) error {
	if err := file.Validate(); err != nil {
		return err
	}

	if gofman.CanUpdateFile(ctx, file) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to create this file.")
	}

	if id, err := tx.db.ID(); err != nil {
		return err
	} else {
		file.ID = id
	}

	file.CreatedAt = tx.now
	file.UpdatedAt = file.CreatedAt

	_, err := tx.ExecContext(ctx, `
		INSERT INTO files (
			id,
			users_id,
			name,
			type,
			path,
			checksum,
			created_at,
			updated_at,
			removed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		file.ID,
		file.UserID,
		file.Name,
		file.Type,
		file.Path,
		file.Checksum,
		file.CreatedAt,
		file.UpdatedAt,
		0,
	)

	if err != nil {
		return err
	}

	return nil
}

// updateFile updates a file object.
// Returns EUNAUTHORIZED if current user is not the creator of the file.
// Returns ENOTFOUND if file does not exist.
func updateFile(ctx context.Context, tx *Tx, id string, update gofman.FileUpdate) (*gofman.File, error) {
	file, err := findFileByID(ctx, tx, id)
	if err != nil {
		return file, err
	}

	if gofman.CanUpdateFile(ctx, file) == false {
		return nil, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to update this file.")
	}

	if v := update.Name; v != nil {
		file.Name = *v
	}

	if v := update.Type; v != nil {
		file.Type = *v
	}

	if v := update.Path; v != nil {
		file.Path = *v
	}

	if v := update.Checksum; v != nil {
		file.Checksum = *v
	}

	file.UpdatedAt = tx.now

	if err := file.Validate(); err != nil {
		return file, err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE files
		SET name = ?,
			type = ?,
			path = ?,
			checksum = ?,
			updated_at = ?
		WHERE id = ?
	`,
		file.Name,
		file.Type,
		file.Path,
		file.Checksum,
		file.UpdatedAt,
		id,
	)

	if err != nil {
		return file, err
	}

	return file, nil
}

// removeFile sets the removed timestamp to the current time. This allows
// us to re-enable removed file.
// Returns EUNAUTHORIZED if current user is not the creator of the file.
// Returns ENOTFOUND if file does not exist.
func removeFile(ctx context.Context, tx *Tx, id string) error {
	file, err := findFileByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if gofman.CanUpdateFile(ctx, file) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to remove this file.")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE files
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
