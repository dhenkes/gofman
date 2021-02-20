package gofman

import (
	"context"
)

// User constants.
const (
	MaxUsernameLen = 35
	MinPasswordLen = 7
)

// User represents a user in the system.
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	IsAdmin   bool   `json:"is_admin"`
	IsDemo    bool   `json:"is_demo"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	RemovedAt int64  `json:"removed_at"`
}

// Validate returns an error if the user contains invalid fields.
func (u *User) Validate() error {
	if u.Username == "" {
		return NewError(EINVALID, "Username required.")
	}

	if len(u.Username) > MaxUsernameLen {
		return NewError(EINVALID, "Username must be less than %d characters.", MaxUsernameLen)
	}

	if u.Password == "" {
		return NewError(EINVALID, "Password required.")
	}

	if len(u.Password) < MinPasswordLen {
		return NewError(EINVALID, "Password must have at least %d characters.", MinPasswordLen)
	}

	return nil
}

// CanFindUser returns true if the current user can list users with
// the given filter.
func CanFindUser(ctx context.Context, filter UserFilter) bool {
	if id := UserIDFromContext(ctx); filter.ID == &id {
		return true
	} else if user := UserFromContext(ctx); user != nil {
		return user.IsAdmin
	} else {
		return false
	}
}

// CanCreateUser returns true if the current user can create a new user.
func CanCreateUser(ctx context.Context) bool {
	if user := UserFromContext(ctx); user != nil {
		return user.IsAdmin
	} else {
		return false
	}
}

// CanUpdateUser returns true if the current user can update the user.
func CanUpdateUser(ctx context.Context, user *User) bool {
	if user := UserFromContext(ctx); user != nil && user.IsDemo {
		return false
	} else if id := UserIDFromContext(ctx); user.ID == id {
		return true
	} else if user := UserFromContext(ctx); user != nil {
		return user.IsAdmin
	} else {
		return false
	}
}

// UserService represents a service for managing users. The functions
// should return ENOTFOUND if the user could not be found and EUNAUTHORIZED
// if the user is not authorized to run the transaction.
type UserService interface {
	FindUserByID(ctx context.Context, id string) (*User, error)
	FindUserByUsername(ctx context.Context, username string) (*User, error)
	FindUsers(ctx context.Context, filter UserFilter) ([]*User, int, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, id string, update UserUpdate) (*User, error)
	RemoveUser(ctx context.Context, id string) error
}

// UserFilter represents a filter passed to FindUsers().
type UserFilter struct {
	ID       *string `json:"id"`
	Username *string `json:"username"`

	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// UserUpdate represents a set of fields to be updated via UpdateUser().
type UserUpdate struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	IsAdmin  *bool   `json:"is_admin"`
}
