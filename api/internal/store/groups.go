package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

type Group struct {
	ID        uuid.UUID
	Name      string
	Currency  string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type Member struct {
	UserID   uuid.UUID
	Name     string
	Email    string
	Role     string
	JoinedAt time.Time
}

const groupColumns = `g.id, g.name, g.currency, g.created_by, g.created_at`

func scanGroup(row pgx.Row) (Group, error) {
	var g Group
	err := row.Scan(&g.ID, &g.Name, &g.Currency, &g.CreatedBy, &g.CreatedAt)
	return g, err
}

// CreateGroup creates a group with owner as its first member.
func (s *Store) CreateGroup(ctx context.Context, name, currency string, owner uuid.UUID) (Group, error) {
	var g Group
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		g, err = scanGroup(tx.QueryRow(ctx,
			`INSERT INTO groups AS g (name, currency, created_by) VALUES ($1, $2, $3)
			 RETURNING `+groupColumns,
			name, currency, owner))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx,
			`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)`,
			g.ID, owner, RoleOwner)
		return err
	})
	if err != nil {
		return Group{}, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// GroupsForUser lists the groups userID belongs to, newest first.
func (s *Store) GroupsForUser(ctx context.Context, userID uuid.UUID) ([]Group, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+groupColumns+`
		 FROM groups g
		 JOIN group_members m ON m.group_id = g.id
		 WHERE m.user_id = $1
		 ORDER BY g.created_at DESC, g.id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	groups, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Group, error) { return scanGroup(r) })
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	return groups, nil
}

func (s *Store) Group(ctx context.Context, id uuid.UUID) (Group, error) {
	g, err := scanGroup(s.pool.QueryRow(ctx, `SELECT `+groupColumns+` FROM groups g WHERE g.id = $1`, id))
	if err != nil {
		return Group{}, fmt.Errorf("get group: %w", notFound(err))
	}
	return g, nil
}

// Members lists a group's members in the order they joined.
func (s *Store) Members(ctx context.Context, groupID uuid.UUID) ([]Member, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id, u.name, u.email, m.role, m.joined_at
		 FROM group_members m
		 JOIN users u ON u.id = m.user_id
		 WHERE m.group_id = $1
		 ORDER BY m.joined_at, u.id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	members, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Member, error) {
		var m Member
		err := r.Scan(&m.UserID, &m.Name, &m.Email, &m.Role, &m.JoinedAt)
		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	return members, nil
}

// MemberRole returns userID's role in the group, or ErrNotMember.
func (s *Store) MemberRole(ctx context.Context, groupID, userID uuid.UUID) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx,
		`SELECT role FROM group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotMember
	}
	if err != nil {
		return "", fmt.Errorf("member role: %w", err)
	}
	return role, nil
}

func (s *Store) AddMember(ctx context.Context, groupID, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO group_members (group_id, user_id, role) VALUES ($1, $2, $3)`,
		groupID, userID, RoleMember)
	if violates(err, codeUniqueViolation, "group_members_pkey") {
		return ErrAlreadyMember
	}
	if err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}
