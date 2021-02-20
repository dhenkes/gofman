package gofman

import (
	"context"
)

// File represents a file in the system.
type File struct {
	ID        string `json:"id"`
	UserID    string `json:"users_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	RemovedAt int64  `json:"removed_at"`
}

// Validate returns an error if the file contains invalid fields.
func (b *File) Validate() error {
	if b.UserID == "" {
		return NewError(EINVALID, "User ID required.")
	}

	if b.Name == "" {
		return NewError(EINVALID, "Name required.")
	}
	if b.Type == "" {
		return NewError(EINVALID, "Type required.")
	}

	if b.Path == "" {
		return NewError(EINVALID, "Path required.")
	}

	if b.Checksum == "" {
		return NewError(EINVALID, "Checksum required.")
	}

	return nil
}

// CanFindFile returns true if the current user can list files with
// the given filter.
func CanFindFile(ctx context.Context, filter FileFilter) bool {
	id := UserIDFromContext(ctx)
	return id != "" && filter.UserID == &id
}

// CanUpdateFile returns true if the current user can update the file.
func CanUpdateFile(ctx context.Context, file *File) bool {
	if user := UserFromContext(ctx); user != nil && user.IsDemo {
		return false
	} else {
		id := UserIDFromContext(ctx)
		return id != "" && file.UserID == id
	}
}

// FileService represents a service for managing files. The functions
// should return ENOTFOUND if the file could not be found and EUNAUTHORIZED
// if the user is not authorized to run the transaction.
type FileService interface {
	FindFileByID(ctx context.Context, id string) (*File, error)
	FindFiles(ctx context.Context, filter FileFilter) ([]*File, int, error)
	CreateFile(ctx context.Context, file *File) error
	UpdateFile(ctx context.Context, id string, update FileUpdate) (*File, error)
	RemoveFile(ctx context.Context, id string) error
}

// FileFilter represents a filter passed to FindFiles().
type FileFilter struct {
	ID     *string `json:"id"`
	UserID *string `json:"users_id"`
	Type   *string `json:"type"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// FileUpdate represents a set of fields to be updated via UpdateFile().
type FileUpdate struct {
	Name     *string `json:"name"`
	Type     *string `json:"type"`
	Path     *string `json:"path"`
	Checksum *string `json:"checksum"`
}
