package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

type SplitMethod string

const (
	SplitEqual   SplitMethod = "equal"
	SplitExact   SplitMethod = "exact"
	SplitPercent SplitMethod = "percent"
	SplitShares  SplitMethod = "shares"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// SplitPart is one participant in a split. Value is ignored for an equal
// split, is an amount in minor units for exact, basis points for percent and
// a relative weight for shares.
type SplitPart struct {
	UserID uuid.UUID
	Value  int64
}

type ExpenseInput struct {
	Description string
	Amount      int64
	PaidBy      uuid.UUID // defaults to the actor
	SpentOn     time.Time // defaults to today
	Method      SplitMethod
	Split       []SplitPart
}

type ExpensePage struct {
	Expenses   []store.Expense
	NextCursor string
}

const notInGroup = "everyone in the split must be a member of the group"

func (s *Service) AddExpense(ctx context.Context, actor, groupID uuid.UUID, in ExpenseInput) (store.Expense, error) {
	if _, err := s.requireMember(ctx, groupID, actor); err != nil {
		return store.Expense{}, err
	}
	members, err := s.store.Members(ctx, groupID)
	if err != nil {
		return store.Expense{}, err
	}
	isMember := make(map[uuid.UUID]bool, len(members))
	for _, m := range members {
		isMember[m.UserID] = true
	}

	in.Description = strings.TrimSpace(in.Description)
	if in.PaidBy == uuid.Nil {
		in.PaidBy = actor
	}
	if in.SpentOn.IsZero() {
		in.SpentOn = s.now()
	}

	v := validator{}
	n := utf8.RuneCountInString(in.Description)
	v.check(n >= 1 && n <= 200, "description", "must be between 1 and 200 characters")
	v.check(in.Amount > 0 && in.Amount <= ledger.MaxAmount, "amount", "must be a positive amount in minor units")
	v.check(isMember[in.PaidBy], "paid_by", "must be a member of the group")
	v.check(validMethod(in.Method), "method", "must be one of equal, exact, percent or shares")
	v.check(len(in.Split) > 0, "split", "must include at least one member")
	for _, p := range in.Split {
		v.check(isMember[p.UserID], "split", notInGroup)
	}
	if err := v.err(); err != nil {
		return store.Expense{}, err
	}

	shares, err := buildShares(in.Amount, in.Method, in.Split)
	if err != nil {
		return store.Expense{}, &ValidationError{Fields: map[string]string{"split": err.Error()}}
	}

	e, err := s.store.CreateExpense(ctx, store.NewExpense{
		GroupID:     groupID,
		PaidBy:      in.PaidBy,
		Description: in.Description,
		Amount:      in.Amount,
		SpentOn:     dateOnly(in.SpentOn),
		CreatedBy:   actor,
		Shares:      shares,
	})
	if errors.Is(err, store.ErrNotMember) {
		return store.Expense{}, &ValidationError{Fields: map[string]string{"split": notInGroup}}
	}
	return e, err
}

// Expenses returns one page of a group's expenses, newest first. cursor is
// the NextCursor from the previous page, or empty for the first page.
func (s *Service) Expenses(ctx context.Context, actor, groupID uuid.UUID, cursor string, limit int) (ExpensePage, error) {
	if _, err := s.requireMember(ctx, groupID, actor); err != nil {
		return ExpensePage{}, err
	}
	if limit == 0 {
		limit = DefaultPageSize
	}
	after, cursorErr := decodeCursor(cursor)

	v := validator{}
	v.check(limit >= 1 && limit <= MaxPageSize, "limit", "must be between 1 and 100")
	v.check(cursorErr == nil, "cursor", "is not a valid page cursor")
	if err := v.err(); err != nil {
		return ExpensePage{}, err
	}

	expenses, next, err := s.store.ListExpenses(ctx, groupID, after, limit)
	if err != nil {
		return ExpensePage{}, err
	}
	return ExpensePage{Expenses: expenses, NextCursor: encodeCursor(next)}, nil
}

// DeleteExpense removes an expense. Its creator and the group owner may do so.
func (s *Service) DeleteExpense(ctx context.Context, actor, groupID, expenseID uuid.UUID) error {
	role, err := s.requireMember(ctx, groupID, actor)
	if err != nil {
		return err
	}
	e, err := s.store.Expense(ctx, groupID, expenseID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if e.CreatedBy != actor && role != store.RoleOwner {
		return ErrForbidden
	}
	err = s.store.DeleteExpense(ctx, groupID, expenseID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func validMethod(m SplitMethod) bool {
	switch m {
	case SplitEqual, SplitExact, SplitPercent, SplitShares:
		return true
	}
	return false
}

func buildShares(amount int64, method SplitMethod, parts []SplitPart) ([]ledger.Share, error) {
	switch method {
	case SplitEqual:
		members := make([]uuid.UUID, len(parts))
		for i, p := range parts {
			members[i] = p.UserID
		}
		return ledger.SplitEqual(amount, members)
	case SplitExact:
		shares := make([]ledger.Share, len(parts))
		for i, p := range parts {
			shares[i] = ledger.Share{Member: p.UserID, Amount: p.Value}
		}
		return ledger.SplitExact(amount, shares)
	case SplitPercent:
		return ledger.SplitPercent(amount, toWeights(parts))
	case SplitShares:
		return ledger.SplitWeighted(amount, toWeights(parts))
	}
	return nil, errors.New("unknown split method")
}

func toWeights(parts []SplitPart) []ledger.Weight {
	weights := make([]ledger.Weight, len(parts))
	for i, p := range parts {
		weights[i] = ledger.Weight{Member: p.UserID, Weight: p.Value}
	}
	return weights
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

var errBadCursor = errors.New("bad cursor")

// Cursors are opaque to clients: base64 of "<created_at>|<id>".
func encodeCursor(c *store.Cursor) string {
	if c == nil {
		return ""
	}
	raw := c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(s string) (*store.Cursor, error) {
	if s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errBadCursor
	}
	ts, id, ok := strings.Cut(string(raw), "|")
	if !ok {
		return nil, errBadCursor
	}
	createdAt, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return nil, errBadCursor
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errBadCursor
	}
	return &store.Cursor{CreatedAt: createdAt, ID: uid}, nil
}
