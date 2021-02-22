package sqlite

import (
	"context"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// Ensure service implements interface.
var _ gofman.UserService = (*UserService)(nil)

// UserService represents a service for managing users.
type UserService struct {
	db *DB
}

// NewUserService returns a new instance of UserService.
func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

// FindUserByID retrieves a user by ID. Returns ENOTFOUND if user does not
// exist.
func (s *UserService) FindUserByID(ctx context.Context, id string) (*gofman.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindUserByUsername retrieves a user by username. Returns ENOTFOUND if user
// does not exist.
func (s *UserService) FindUserByUsername(ctx context.Context, username string) (*gofman.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := findUserByUsername(ctx, tx, username)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindUsers retrieves users and total hits based on a filter.
func (s *UserService) FindUsers(ctx context.Context, filter gofman.UserFilter) ([]*gofman.User, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback()

	users, total, err := findUsers(ctx, tx, filter)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// CreateUser creates a new user.
func (s *UserService) CreateUser(ctx context.Context, user *gofman.User) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createUser(ctx, tx, user); err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateUser updates a user. Returns EUNAUTHORIZED if current user is not
// user being updated. Returns ENOTFOUND if user does not exist.
func (s *UserService) UpdateUser(ctx context.Context, id string, update gofman.UserUpdate) (*gofman.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := updateUser(ctx, tx, id, update)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

// RemoveUser sets the removed timestamp to the current time. Returns
// EUNAUTHORIZED if current user is not the user being removed. Returns
// ENOTFOUND if user does not exist.
func (s *UserService) RemoveUser(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := removeUser(ctx, tx, id); err != nil {
		return err
	}

	return tx.Commit()
}

// findUserByID is a helper function to fetch a user by ID.
// Returns ENOTFOUND if user does not exist.
func findUserByID(ctx context.Context, tx *Tx, id string) (*gofman.User, error) {
	users, _, err := findUsers(ctx, tx, gofman.UserFilter{ID: &id, Limit: 1})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "User not found.")
	}

	return users[0], nil
}

// findUserByUsername is a helper function to fetch a user by ID.
// Returns ENOTFOUND if user does not exist.
func findUserByUsername(ctx context.Context, tx *Tx, username string) (*gofman.User, error) {
	users, _, err := findUsers(ctx, tx, gofman.UserFilter{Username: &username, Limit: 1})
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, gofman.NewError(gofman.ENOTFOUND, "User not found.")
	}

	return users[0], nil
}

// findUsers returns a list of users matching a filter.
func findUsers(ctx context.Context, tx *Tx, filter gofman.UserFilter) ([]*gofman.User, int, error) {
	if gofman.CanFindUser(ctx, filter) == false {
		return nil, 0, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to search using this filter.")
	}

	where, args := []string{"1 = 1"}, []interface{}{}

	if v := filter.ID; v != nil {
		where, args = append(where, "id = ?"), append(args, *v)
	}

	if v := filter.Username; v != nil {
		where, args = append(where, "username = ?"), append(args, *v)
	}

	where = append(where, "removed_at = 0")

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			username,
			password,
			is_admin,
			created_at,
			updated_at,
			removed_at,
			COUNT(*) OVER()
		FROM users
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
	var users []*gofman.User

	for rows.Next() {
		var user gofman.User

		if err = rows.Scan(
			&user.ID, &user.Username, &user.Password, &user.IsAdmin,
			&user.CreatedAt, &user.UpdatedAt, &user.RemovedAt,
			&n,
		); err != nil {
			return nil, 0, err
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, n, nil
}

// createUser creates a new user.
func createUser(ctx context.Context, tx *Tx, user *gofman.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	if gofman.CanCreateUser(ctx) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to create this user.")
	}

	if id, err := tx.db.ID(); err != nil {
		return err
	} else {
		user.ID = id
	}

	if hash, err := hashPassword(ctx, tx, user.Password); err != nil {
		return err
	} else {
		user.Password = hash
	}

	user.Username = strings.ToLower(user.Username)
	user.IsAdmin = false
	user.CreatedAt = tx.now
	user.UpdatedAt = user.CreatedAt

	_, err := tx.ExecContext(ctx, `
		INSERT INTO users (
			id,
			username,
			password,
			is_admin,
			created_at,
			updated_at,
			removed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`,
		user.ID,
		user.Username,
		user.Password,
		user.IsAdmin,
		user.CreatedAt,
		user.UpdatedAt,
		0,
	)

	if err != nil {
		return err
	}

	return nil
}

// updateUser updates a user. Returns EUNAUTHORIZED if current user is not
// user being updated. Returns ENOTFOUND if user does not exist.
func updateUser(ctx context.Context, tx *Tx, id string, update gofman.UserUpdate) (*gofman.User, error) {
	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return user, err
	}

	if gofman.CanUpdateUser(ctx, user) == false {
		return nil, gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to update this user.")
	}

	if v := update.Username; v != nil {
		user.Username = *v
	}

	if v := update.Password; v != nil {
		user.Password = *v
	}

	if v := update.IsAdmin; v != nil {
		user.IsAdmin = *v
	}

	user.UpdatedAt = tx.now

	if err := user.Validate(); err != nil {
		return user, err
	}

	user.Username = strings.ToLower(user.Username)

	if v := update.Password; v != nil {
		if user.Password, err = hashPassword(ctx, tx, user.Password); err != nil {
			return nil, err
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET username = ?,
			password = ?,
			is_admin = ?,
			updated_at = ?
		WHERE id = ?
	`,
		user.Username,
		user.Password,
		user.IsAdmin,
		user.UpdatedAt,
		id,
	)

	if err != nil {
		return user, err
	}

	return user, nil
}

// removeUser sets the removed timestamp to the current time. Returns
// EUNAUTHORIZED if current user is not the user being removed. Returns
// ENOTFOUND if user does not exist.
func removeUser(ctx context.Context, tx *Tx, id string) error {
	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if gofman.CanUpdateUser(ctx, user) == false {
		return gofman.NewError(gofman.EUNAUTHORIZED, "You are not allowed to remove this user.")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users
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

// hashPassword is a helper function that takes a password, generates a salt
// and returns the hashed password or an error.
func hashPassword(ctx context.Context, tx *Tx, password string) (string, error) {
	if tx.db.AuthService == nil {
		return "", gofman.NewError(gofman.EINVALID, "AuthService required.")
	}

	salt, err := tx.db.AuthService.NewSalt()
	if err != nil {
		return "", err
	}

	hash, err := tx.db.AuthService.HashPassword(password, salt)
	if err != nil {
		return "", err
	}

	return hash, nil
}
