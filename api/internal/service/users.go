package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

// Session is what a client receives after registering or logging in.
type Session struct {
	User      store.User
	Token     string
	ExpiresAt time.Time
}

type RegisterInput struct {
	Email    string
	Name     string
	Password string
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Session, error) {
	in.Email = strings.TrimSpace(in.Email)
	in.Name = strings.TrimSpace(in.Name)

	v := validator{}
	v.check(validEmail(in.Email), "email", "must be a valid email address")
	v.check(validName(in.Name), "name", "must be between 1 and 100 characters")
	v.check(len(in.Password) >= auth.MinPasswordLength, "password", "must be at least 8 characters")
	v.check(len(in.Password) <= auth.MaxPasswordLength, "password", "must be at most 72 bytes")
	if err := v.err(); err != nil {
		return Session{}, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return Session{}, err
	}
	user, err := s.store.CreateUser(ctx, in.Email, in.Name, hash)
	if errors.Is(err, store.ErrEmailTaken) {
		return Session{}, ErrEmailTaken
	}
	if err != nil {
		return Session{}, err
	}
	return s.newSession(user)
}

func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := s.store.UserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, store.ErrNotFound) {
		s.hasher.Check(s.dummyHash, password)
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	if !s.hasher.Check(user.PasswordHash, password) {
		return Session{}, ErrInvalidCredentials
	}
	return s.newSession(user)
}

// Authenticate turns a bearer token into the ID of the user it belongs to.
func (s *Service) Authenticate(token string) (uuid.UUID, error) {
	return s.tokens.Verify(token)
}

func (s *Service) User(ctx context.Context, id uuid.UUID) (store.User, error) {
	user, err := s.store.UserByID(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return store.User{}, ErrNotFound
	}
	return user, err
}

func (s *Service) newSession(user store.User) (Session, error) {
	token, expiresAt, err := s.tokens.Issue(user.ID)
	if err != nil {
		return Session{}, err
	}
	return Session{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

func validEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	// ParseAddress also accepts "Name <a@b.c>"; we only want the bare address.
	return err == nil && addr.Address == email
}

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n >= 1 && n <= 100
}
