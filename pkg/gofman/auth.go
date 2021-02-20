package gofman

// AuthService represents a service for managing authentication. It should be
// used for creating, hasing and comparing passwords and tokens.
type AuthService interface {
	NewToken() (string, error)
	NewPassword() (string, error)
	NewSalt() (string, error)
	HashPassword(password string, salt string) (string, error)
	VerifyPassword(password string, hash string) error
}
