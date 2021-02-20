package gofman

import (
	"context"
)

// Tag constants.
const (
	MaxTagNameLen = 255
)

// Tag represents a tag in the system.
type Tag struct {
	ID        string `json:"id"`
	UserID    string `json:"users_id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	RemovedAt int64  `json:"removed_at"`
}

// Validate returns an error if the tag contains invalid fields.
func (t *Tag) Validate() error {
	if t.UserID == "" {
		return NewError(EINVALID, "User ID required.")
	}

	if t.Name == "" {
		return NewError(EINVALID, "Name required.")
	}

	if len(t.Name) > MaxTagNameLen {
		return NewError(EINVALID, "Name must be less than %d characters.", MaxTagNameLen)
	}

	return nil
}

// CanFindTag returns true if the current user can list tags with
// the given filter.
func CanFindTag(ctx context.Context, filter TagFilter) bool {
	id := UserIDFromContext(ctx)
	return id != "" && filter.UserID == &id
}

// CanUpdateTag returns true if the current user can update the tag.
func CanUpdateTag(ctx context.Context, tag *Tag) bool {
	if user := UserFromContext(ctx); user != nil && user.IsDemo {
		return false
	} else {
		id := UserIDFromContext(ctx)
		return id != "" && tag.UserID == id
	}
}

// TagService represents a service for managing tags. The functions
// should return ENOTFOUND if the tag could not be found and EUNAUTHORIZED
// if the user is not authorized to run the transaction.
type TagService interface {
	FindTagByID(ctx context.Context, id string) (*Tag, error)
	FindTags(ctx context.Context, filter TagFilter) ([]*Tag, int, error)
	CreateTag(ctx context.Context, tag *Tag) error
	UpdateTag(ctx context.Context, id string, update TagUpdate) (*Tag, error)
	RemoveTag(ctx context.Context, id string) error
}

// TagFilter represents a filter passed to FindTags().
type TagFilter struct {
	ID     *string `json:"id"`
	UserID *string `json:"users_id"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// TagUpdate represents a set of fields to be updated via UpdateTag().
type TagUpdate struct {
	Name *string `json:"name"`
}
