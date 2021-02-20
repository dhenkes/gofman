package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dhenkes/gofman/pkg/gofman"
	"golang.org/x/crypto/argon2"
)

// Auth constants.
const (
	ArgonTime    = 1
	ArgonMemory  = 64 * 1024
	ArgonThreads = 4
	ArgonKeyLen  = 32
)

// ArgonSettings is used to extract the basic hash settings from a string.
type ArgonSettings struct {
	Version int
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// Ensure service implements interface.
var _ gofman.AuthService = (*AuthService)(nil)

// AuthService represents a service for managing authentication.
type AuthService struct{}

// NewAuthService returns a new instance of AuthService.
func NewAuthService() *AuthService {
	return &AuthService{}
}

// GenerateRandomBytes is a helper function that is used by NewToken,
// NewPassword and NewSalt. It returns securely generated random bytes.
func GenerateRandomBytes(n int) ([]byte, error) {
	if n < -1 {
		return nil, gofman.NewError(gofman.EINTERNAL, "Length must be a positive int.")
	}

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	} else {
		return b, nil
	}
}

// EncodeToBase64String is a helper function that turns the given bytes into
// a base64 encoded string.
func EncodeToBase64String(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// DecodeBase64String is a helper function that decodes the given base64 string.
func DecodeBase64String(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// NewToken generates a new token that can be used as a session-key.
func (s *AuthService) NewToken() (string, error) {
	if b, err := GenerateRandomBytes(32); err != nil {
		return "", err
	} else {
		return EncodeToBase64String(b), nil
	}
}

// NewPassword is meant to generate temporary passwords if a user does not
// supply one on his own.
func (s *AuthService) NewPassword() (string, error) {
	if b, err := GenerateRandomBytes(8); err != nil {
		return "", err
	} else {
		return EncodeToBase64String(b), nil
	}
}

// NewSalt generates a secure salt that can be used in combination with the
// HashPassword function.
func (s *AuthService) NewSalt() (string, error) {
	if b, err := GenerateRandomBytes(16); err != nil {
		return "", err
	} else {
		return EncodeToBase64String(b), nil
	}
}

// HashPassword takes a password and a salt and returns an argon2 key that
// can be saved in a database.
func (s *AuthService) HashPassword(password string, salt string) (string, error) {
	if password == "" {
		return "", gofman.NewError(gofman.EINVALID, "Password required.")
	}

	if salt == "" {
		return "", gofman.NewError(gofman.EINVALID, "Salt required.")
	}

	hash := argon2.IDKey(
		[]byte(password), []byte(salt),
		ArgonTime, ArgonMemory, ArgonThreads, ArgonKeyLen,
	)

	b64Salt := EncodeToBase64String([]byte(salt))
	b64Hash := EncodeToBase64String(hash)

	key := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, ArgonMemory, ArgonTime, ArgonThreads, b64Salt, b64Hash,
	)

	return key, nil
}

// VerifyPassword takes a password and an argon2 key and compares both. It will
// return an error if they are not equal.
func (s *AuthService) VerifyPassword(password string, key string) error {
	if password == "" {
		return gofman.NewError(gofman.EINVALID, "Password required.")
	}

	if key == "" {
		return gofman.NewError(gofman.EINVALID, "Argon2 key required.")
	}

	decodedKey := strings.Split(key, "$")
	if len(decodedKey) != 6 {
		return gofman.NewError(gofman.EINVALID, "Decoded key wrong length.")
	}

	p := ArgonSettings{}

	if _, err := fmt.Sscanf(decodedKey[2], "v=%d", &p.Version); err != nil {
		return err
	}

	if p.Version != argon2.Version {
		return gofman.NewError(gofman.EINVALID, "Argon version mismatch.")
	}

	if _, err := fmt.Sscanf(decodedKey[3], "m=%d,t=%d,p=%d",
		&p.Memory, &p.Time, &p.Threads,
	); err != nil {
		return err
	}

	salt, err := DecodeBase64String(decodedKey[4])
	if err != nil {
		return err
	}

	hash, err := DecodeBase64String(decodedKey[5])
	if err != nil {
		return err
	}

	p.KeyLen = uint32(len(hash))

	control := argon2.IDKey(
		[]byte(password), []byte(salt),
		p.Time, p.Memory, p.Threads, p.KeyLen,
	)

	if subtle.ConstantTimeCompare(hash, control) == 1 {
		return nil
	} else {
		return gofman.NewError(gofman.EINVALID, "Hash not equal password.")
	}
}
