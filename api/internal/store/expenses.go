package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
)

type Expense struct {
	ID          uuid.UUID
	GroupID     uuid.UUID
	PaidBy      uuid.UUID
	Description string
	Amount      int64
	SpentOn     time.Time
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
	Shares      []ledger.Share
}

type NewExpense struct {
	GroupID     uuid.UUID
	PaidBy      uuid.UUID
	Description string
	Amount      int64
	SpentOn     time.Time
	CreatedBy   uuid.UUID
	Shares      []ledger.Share
}

// Cursor marks a position in a group's expense list, which is ordered by
// creation time and then ID, newest first.
type Cursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

const expenseColumns = `id, group_id, paid_by, description, amount, spent_on, created_by, created_at`

func scanExpense(row pgx.Row) (Expense, error) {
	var e Expense
	err := row.Scan(&e.ID, &e.GroupID, &e.PaidBy, &e.Description, &e.Amount, &e.SpentOn, &e.CreatedBy, &e.CreatedAt)
	return e, err
}

// CreateExpense stores an expense and its shares in one transaction. It
// returns ErrNotMember if the payer or anyone in the split is not in the group.
func (s *Store) CreateExpense(ctx context.Context, n NewExpense) (Expense, error) {
	var e Expense
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		e, err = scanExpense(tx.QueryRow(ctx,
			`INSERT INTO expenses (group_id, paid_by, description, amount, spent_on, created_by)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING `+expenseColumns,
			n.GroupID, n.PaidBy, n.Description, n.Amount, n.SpentOn, n.CreatedBy))
		if err != nil {
			return err
		}
		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"expense_shares"},
			[]string{"expense_id", "group_id", "user_id", "amount"},
			pgx.CopyFromSlice(len(n.Shares), func(i int) ([]any, error) {
				return []any{e.ID, n.GroupID, n.Shares[i].Member, n.Shares[i].Amount}, nil
			}))
		return err
	})
	if violates(err, codeForeignKeyViolation, "") {
		return Expense{}, ErrNotMember
	}
	if err != nil {
		return Expense{}, fmt.Errorf("create expense: %w", err)
	}
	e.Shares = n.Shares
	return e, nil
}

// ListExpenses returns up to limit expenses older than after, newest first,
// plus a cursor for the next page if there is one.
func (s *Store) ListExpenses(ctx context.Context, groupID uuid.UUID, after *Cursor, limit int) ([]Expense, *Cursor, error) {
	var afterTime *time.Time
	afterID := uuid.Nil
	if after != nil {
		afterTime, afterID = &after.CreatedAt, after.ID
	}

	rows, err := s.pool.Query(ctx,
		`SELECT `+expenseColumns+`
		 FROM expenses
		 WHERE group_id = $1
		   AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3::uuid))
		 ORDER BY created_at DESC, id DESC
		 LIMIT $4`,
		groupID, afterTime, afterID, limit+1)
	if err != nil {
		return nil, nil, fmt.Errorf("list expenses: %w", err)
	}
	expenses, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Expense, error) { return scanExpense(r) })
	if err != nil {
		return nil, nil, fmt.Errorf("list expenses: %w", err)
	}

	var next *Cursor
	if len(expenses) > limit {
		expenses = expenses[:limit]
		last := expenses[limit-1]
		next = &Cursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if err := s.attachShares(ctx, expenses); err != nil {
		return nil, nil, err
	}
	return expenses, next, nil
}

// Expense returns one expense, scoped to its group so an ID from another
// group reads as not found.
func (s *Store) Expense(ctx context.Context, groupID, id uuid.UUID) (Expense, error) {
	e, err := scanExpense(s.pool.QueryRow(ctx,
		`SELECT `+expenseColumns+` FROM expenses WHERE group_id = $1 AND id = $2`, groupID, id))
	if err != nil {
		return Expense{}, fmt.Errorf("get expense: %w", notFound(err))
	}
	list := []Expense{e}
	if err := s.attachShares(ctx, list); err != nil {
		return Expense{}, err
	}
	return list[0], nil
}

func (s *Store) DeleteExpense(ctx context.Context, groupID, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM expenses WHERE group_id = $1 AND id = $2`, groupID, id)
	if err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// attachShares loads the shares for every expense with a single query.
func (s *Store) attachShares(ctx context.Context, expenses []Expense) error {
	if len(expenses) == 0 {
		return nil
	}
	index := make(map[uuid.UUID]int, len(expenses))
	ids := make([]uuid.UUID, len(expenses))
	for i, e := range expenses {
		index[e.ID] = i
		ids[i] = e.ID
	}

	rows, err := s.pool.Query(ctx,
		`SELECT s.expense_id, s.user_id, s.amount
		 FROM expense_shares s
		 JOIN group_members m ON m.group_id = s.group_id AND m.user_id = s.user_id
		 WHERE s.expense_id = ANY($1)
		 ORDER BY m.joined_at, s.user_id`, ids)
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	var expenseID uuid.UUID
	var share ledger.Share
	_, err = pgx.ForEachRow(rows, []any{&expenseID, &share.Member, &share.Amount}, func() error {
		i := index[expenseID]
		expenses[i].Shares = append(expenses[i].Shares, share)
		return nil
	})
	if err != nil {
		return fmt.Errorf("load shares: %w", err)
	}
	return nil
}
