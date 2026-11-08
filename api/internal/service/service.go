// Package service holds the business rules: input validation, who may see or
// change what, and turning split requests into exact shares.
package service

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("you do not have permission to do that")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailTaken         = errors.New("an account with this email already exists")
	ErrAlreadyMember      = errors.New("that user is already in the group")
)

// ValidationError reports problems with user input, keyed by field name.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for _, k := range slices.Sorted(maps.Keys(e.Fields)) {
		parts = append(parts, fmt.Sprintf("%s: %s", k, e.Fields[k]))
	}
	return "invalid input: " + strings.Join(parts, "; ")
}

// validator collects field errors so a request reports all of them at once.
type validator map[string]string

func (v validator) check(ok bool, field, msg string) {
	if !ok {
		if _, exists := v[field]; !exists {
			v[field] = msg
		}
	}
}

func (v validator) err() error {
	if len(v) == 0 {
		return nil
	}
	return &ValidationError{Fields: v}
}

type Service struct {
	store     *store.Store
	hasher    auth.Hasher
	tokens    *auth.Tokens
	dummyHash string
	now       func() time.Time
}

func New(st *store.Store, hasher auth.Hasher, tokens *auth.Tokens) (*Service, error) {
	// Logging in with an unknown email still runs one bcrypt comparison
	// against this hash, so response time does not reveal which emails exist.
	dummy, err := hasher.Hash("fairshare-timing-equalizer")
	if err != nil {
		return nil, err
	}
	return &Service{store: st, hasher: hasher, tokens: tokens, dummyHash: dummy, now: time.Now}, nil
}
