package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/store"
)

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

type GroupDetail struct {
	Group   store.Group
	Members []store.Member
}

func (s *Service) CreateGroup(ctx context.Context, actor uuid.UUID, name, currency string) (GroupDetail, error) {
	name = strings.TrimSpace(name)
	currency = strings.ToUpper(strings.TrimSpace(currency))

	v := validator{}
	v.check(validName(name), "name", "must be between 1 and 100 characters")
	v.check(currencyCode.MatchString(currency), "currency", "must be a three-letter ISO 4217 code such as USD")
	if err := v.err(); err != nil {
		return GroupDetail{}, err
	}

	g, err := s.store.CreateGroup(ctx, name, currency, actor)
	if err != nil {
		return GroupDetail{}, err
	}
	return s.groupDetail(ctx, g)
}

func (s *Service) Groups(ctx context.Context, actor uuid.UUID) ([]store.Group, error) {
	return s.store.GroupsForUser(ctx, actor)
}

func (s *Service) Group(ctx context.Context, actor, groupID uuid.UUID) (GroupDetail, error) {
	if _, err := s.requireMember(ctx, groupID, actor); err != nil {
		return GroupDetail{}, err
	}
	g, err := s.store.Group(ctx, groupID)
	if err != nil {
		return GroupDetail{}, err
	}
	return s.groupDetail(ctx, g)
}

// AddMember adds an existing user to the group by email. Only the owner may
// add people.
func (s *Service) AddMember(ctx context.Context, actor, groupID uuid.UUID, email string) (GroupDetail, error) {
	role, err := s.requireMember(ctx, groupID, actor)
	if err != nil {
		return GroupDetail{}, err
	}
	if role != store.RoleOwner {
		return GroupDetail{}, ErrForbidden
	}

	user, err := s.store.UserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, store.ErrNotFound) {
		return GroupDetail{}, &ValidationError{Fields: map[string]string{"email": "no account uses this email"}}
	}
	if err != nil {
		return GroupDetail{}, err
	}
	err = s.store.AddMember(ctx, groupID, user.ID)
	if errors.Is(err, store.ErrAlreadyMember) {
		return GroupDetail{}, ErrAlreadyMember
	}
	if err != nil {
		return GroupDetail{}, err
	}
	return s.Group(ctx, actor, groupID)
}

func (s *Service) groupDetail(ctx context.Context, g store.Group) (GroupDetail, error) {
	members, err := s.store.Members(ctx, g.ID)
	if err != nil {
		return GroupDetail{}, err
	}
	return GroupDetail{Group: g, Members: members}, nil
}

// requireMember returns the actor's role in the group. Groups the actor is
// not in look exactly like groups that do not exist, so IDs cannot be probed.
func (s *Service) requireMember(ctx context.Context, groupID, actor uuid.UUID) (string, error) {
	role, err := s.store.MemberRole(ctx, groupID, actor)
	if errors.Is(err, store.ErrNotMember) {
		return "", ErrNotFound
	}
	return role, err
}
