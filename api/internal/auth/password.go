// Package auth hashes passwords and issues and verifies access tokens.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	MinPasswordLength = 8
	// MaxPasswordLength is in bytes. bcrypt silently ignores anything past 72.
	MaxPasswordLength = 72

	// DefaultCost is two steps above bcrypt's own default of 10.
	DefaultCost = 12
)

// Hasher hashes and checks passwords with bcrypt at a fixed cost. Production
// uses DefaultCost; tests use bcrypt.MinCost to stay fast.
type Hasher struct {
	cost int
}

func NewHasher(cost int) Hasher {
	return Hasher{cost: cost}
}

func (h Hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// Check reports whether password matches hash.
func (h Hasher) Check(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
