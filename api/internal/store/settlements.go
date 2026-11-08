package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Settlement struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	From      uuid.UUID
	To        uuid.UUID
	Amount    int64
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type NewSettlement struct {
	GroupID   uuid.UUID
	From      uuid.UUID
	To        uuid.UUID
	Amount    int64
	CreatedBy uuid.UUID
}

// MemberBalance is a member's net position. Positive means the group owes
// them, negative means they owe the group.
type MemberBalance struct {
	UserID uuid.UUID
	Name   string
	Net    int64
}

const settlementColumns = `id, group_id, from_user, to_user, amount, created_by, created_at`

func scanSettlement(row pgx.Row) (Settlement, error) {
	var st Settlement
	err := row.Scan(&st.ID, &st.GroupID, &st.From, &st.To, &st.Amount, &st.CreatedBy, &st.CreatedAt)
	return st, err
}

// CreateSettlement records a payment between two members of a group.
func (s *Store) CreateSettlement(ctx context.Context, n NewSettlement) (Settlement, error) {
	st, err := scanSettlement(s.pool.QueryRow(ctx,
		`INSERT INTO settlements (group_id, from_user, to_user, amount, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+settlementColumns,
		n.GroupID, n.From, n.To, n.Amount, n.CreatedBy))
	if violates(err, codeForeignKeyViolation, "") {
		return Settlement{}, ErrNotMember
	}
	if err != nil {
		return Settlement{}, fmt.Errorf("create settlement: %w", err)
	}
	return st, nil
}

// ListSettlements returns a group's settlements, newest first.
func (s *Store) ListSettlements(ctx context.Context, groupID uuid.UUID) ([]Settlement, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+settlementColumns+`
		 FROM settlements
		 WHERE group_id = $1
		 ORDER BY created_at DESC, id DESC`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list settlements: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Settlement, error) { return scanSettlement(r) })
	if err != nil {
		return nil, fmt.Errorf("list settlements: %w", err)
	}
	return list, nil
}

// Balances computes every member's net position from expenses and
// settlements. The aggregation runs in Postgres so the cost does not grow
// with the number of rows sent to the application.
func (s *Store) Balances(ctx context.Context, groupID uuid.UUID) ([]MemberBalance, error) {
	rows, err := s.pool.Query(ctx,
		`WITH paid AS (
		     SELECT paid_by AS user_id, sum(amount) AS total
		     FROM expenses WHERE group_id = $1 GROUP BY paid_by
		 ), owed AS (
		     SELECT user_id, sum(amount) AS total
		     FROM expense_shares WHERE group_id = $1 GROUP BY user_id
		 ), sent AS (
		     SELECT from_user AS user_id, sum(amount) AS total
		     FROM settlements WHERE group_id = $1 GROUP BY from_user
		 ), received AS (
		     SELECT to_user AS user_id, sum(amount) AS total
		     FROM settlements WHERE group_id = $1 GROUP BY to_user
		 )
		 SELECT m.user_id, u.name,
		        (coalesce(p.total, 0) - coalesce(o.total, 0)
		         + coalesce(s.total, 0) - coalesce(r.total, 0))::bigint AS net
		 FROM group_members m
		 JOIN users u ON u.id = m.user_id
		 LEFT JOIN paid p ON p.user_id = m.user_id
		 LEFT JOIN owed o ON o.user_id = m.user_id
		 LEFT JOIN sent s ON s.user_id = m.user_id
		 LEFT JOIN received r ON r.user_id = m.user_id
		 WHERE m.group_id = $1
		 ORDER BY m.joined_at, m.user_id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("balances: %w", err)
	}
	balances, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (MemberBalance, error) {
		var b MemberBalance
		err := r.Scan(&b.UserID, &b.Name, &b.Net)
		return b, err
	})
	if err != nil {
		return nil, fmt.Errorf("balances: %w", err)
	}
	return balances, nil
}
