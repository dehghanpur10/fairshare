package store_test

import (
	"bytes"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
	"github.com/dehghanpur10/fairshare/api/internal/store"
	"github.com/dehghanpur10/fairshare/api/internal/testdb"
)

func newStore(t *testing.T) *store.Store {
	t.Parallel()
	return store.New(testdb.New(t))
}

func mustUser(t *testing.T, s *store.Store, email, name string) store.User {
	t.Helper()
	u, err := s.CreateUser(t.Context(), email, name, "hash")
	if err != nil {
		t.Fatalf("CreateUser(%s): %v", email, err)
	}
	return u
}

// seedGroup creates a group owned by the first user with the rest added as
// members.
func seedGroup(t *testing.T, s *store.Store, users ...store.User) store.Group {
	t.Helper()
	g, err := s.CreateGroup(t.Context(), "Lisbon trip", "EUR", users[0].ID)
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	for _, u := range users[1:] {
		if err := s.AddMember(t.Context(), g.ID, u.ID); err != nil {
			t.Fatalf("AddMember: %v", err)
		}
	}
	return g
}

func TestUsers(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()

	ana := mustUser(t, s, "Ana@Example.com", "Ana")
	if _, err := s.CreateUser(ctx, "ana@example.COM", "Ana again", "hash"); !errors.Is(err, store.ErrEmailTaken) {
		t.Errorf("duplicate email in another case: got %v, want ErrEmailTaken", err)
	}

	got, err := s.UserByEmail(ctx, "ANA@EXAMPLE.COM")
	if err != nil || got.ID != ana.ID {
		t.Errorf("UserByEmail: got %v, %v; want %v", got.ID, err, ana.ID)
	}
	if _, err := s.UserByID(ctx, uuid.New()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown id: got %v, want ErrNotFound", err)
	}
}

func TestGroupsAndMembers(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	ana, ben, cat := mustUser(t, s, "ana@example.com", "Ana"), mustUser(t, s, "ben@example.com", "Ben"), mustUser(t, s, "cat@example.com", "Cat")

	g := seedGroup(t, s, ana, ben)

	if role, err := s.MemberRole(ctx, g.ID, ana.ID); err != nil || role != store.RoleOwner {
		t.Errorf("creator role: got %q, %v; want owner", role, err)
	}
	if role, err := s.MemberRole(ctx, g.ID, ben.ID); err != nil || role != store.RoleMember {
		t.Errorf("added member role: got %q, %v; want member", role, err)
	}
	if _, err := s.MemberRole(ctx, g.ID, cat.ID); !errors.Is(err, store.ErrNotMember) {
		t.Errorf("outsider: got %v, want ErrNotMember", err)
	}
	if err := s.AddMember(ctx, g.ID, ben.ID); !errors.Is(err, store.ErrAlreadyMember) {
		t.Errorf("adding twice: got %v, want ErrAlreadyMember", err)
	}

	members, err := s.Members(ctx, g.ID)
	if err != nil || len(members) != 2 || members[0].UserID != ana.ID {
		t.Errorf("Members: got %+v, %v", members, err)
	}

	groups, err := s.GroupsForUser(ctx, cat.ID)
	if err != nil || len(groups) != 0 {
		t.Errorf("outsider's groups: got %+v, %v; want none", groups, err)
	}
}

func TestExpensesRejectOutsiders(t *testing.T) {
	s := newStore(t)
	ana, cat := mustUser(t, s, "ana@example.com", "Ana"), mustUser(t, s, "cat@example.com", "Cat")
	g := seedGroup(t, s, ana)

	_, err := s.CreateExpense(t.Context(), store.NewExpense{
		GroupID: g.ID, PaidBy: ana.ID, CreatedBy: ana.ID,
		Description: "Dinner", Amount: 1000, SpentOn: time.Now(),
		Shares: []ledger.Share{{Member: ana.ID, Amount: 500}, {Member: cat.ID, Amount: 500}},
	})
	if !errors.Is(err, store.ErrNotMember) {
		t.Fatalf("split with an outsider: got %v, want ErrNotMember", err)
	}

	page, _, err := s.ListExpenses(t.Context(), g.ID, nil, 10)
	if err != nil || len(page) != 0 {
		t.Errorf("failed insert left rows behind: %+v, %v", page, err)
	}
}

func TestExpensePagination(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	ana, ben := mustUser(t, s, "ana@example.com", "Ana"), mustUser(t, s, "ben@example.com", "Ben")
	g := seedGroup(t, s, ana, ben)

	created := make(map[uuid.UUID]bool)
	for i := range 5 {
		e, err := s.CreateExpense(ctx, store.NewExpense{
			GroupID: g.ID, PaidBy: ana.ID, CreatedBy: ana.ID,
			Description: "Taxi", Amount: int64(100 * (i + 1)), SpentOn: time.Now(),
			Shares: []ledger.Share{{Member: ana.ID, Amount: int64(50 * (i + 1))}, {Member: ben.ID, Amount: int64(50 * (i + 1))}},
		})
		if err != nil {
			t.Fatalf("CreateExpense: %v", err)
		}
		created[e.ID] = true
	}

	var seen []store.Expense
	var cursor *store.Cursor
	for pages := 0; ; pages++ {
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
		page, next, err := s.ListExpenses(ctx, g.ID, cursor, 2)
		if err != nil {
			t.Fatalf("ListExpenses: %v", err)
		}
		for _, e := range page {
			if len(e.Shares) != 2 || e.Shares[0].Member != ana.ID {
				t.Errorf("expense %v has shares %+v", e.ID, e.Shares)
			}
			if !created[e.ID] {
				t.Errorf("expense %v returned twice or never created", e.ID)
			}
			delete(created, e.ID)
			seen = append(seen, e)
		}
		if next == nil {
			break
		}
		cursor = next
	}

	if len(created) != 0 {
		t.Errorf("%d expenses never appeared in any page", len(created))
	}
	newestFirst := slices.IsSortedFunc(seen, func(a, b store.Expense) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return bytes.Compare(b.ID[:], a.ID[:])
	})
	if !newestFirst {
		t.Error("pages are not ordered newest first")
	}
}

func TestBalancesAndSettlements(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	ana, ben, cat := mustUser(t, s, "ana@example.com", "Ana"), mustUser(t, s, "ben@example.com", "Ben"), mustUser(t, s, "cat@example.com", "Cat")
	g := seedGroup(t, s, ana, ben, cat)

	// Ana pays 90.00 for everyone, Ben pays 30.00 for himself and Cat.
	for _, e := range []store.NewExpense{
		{PaidBy: ana.ID, Amount: 9000, Shares: []ledger.Share{{Member: ana.ID, Amount: 3000}, {Member: ben.ID, Amount: 3000}, {Member: cat.ID, Amount: 3000}}},
		{PaidBy: ben.ID, Amount: 3000, Shares: []ledger.Share{{Member: ben.ID, Amount: 1500}, {Member: cat.ID, Amount: 1500}}},
	} {
		e.GroupID, e.CreatedBy, e.Description, e.SpentOn = g.ID, e.PaidBy, "Groceries", time.Now()
		if _, err := s.CreateExpense(ctx, e); err != nil {
			t.Fatalf("CreateExpense: %v", err)
		}
	}
	if _, err := s.CreateSettlement(ctx, store.NewSettlement{GroupID: g.ID, From: cat.ID, To: ana.ID, Amount: 2000, CreatedBy: cat.ID}); err != nil {
		t.Fatalf("CreateSettlement: %v", err)
	}

	balances, err := s.Balances(ctx, g.ID)
	if err != nil {
		t.Fatalf("Balances: %v", err)
	}
	want := map[uuid.UUID]int64{ana.ID: 4000, ben.ID: -1500, cat.ID: -2500}
	var sum int64
	for _, b := range balances {
		sum += b.Net
		if b.Net != want[b.UserID] {
			t.Errorf("%s: net %d, want %d", b.Name, b.Net, want[b.UserID])
		}
	}
	if sum != 0 {
		t.Errorf("balances sum to %d, want 0", sum)
	}

	if _, err := s.CreateSettlement(ctx, store.NewSettlement{GroupID: g.ID, From: ana.ID, To: uuid.New(), Amount: 1, CreatedBy: ana.ID}); !errors.Is(err, store.ErrNotMember) {
		t.Errorf("settlement with an outsider: got %v, want ErrNotMember", err)
	}
	list, err := s.ListSettlements(ctx, g.ID)
	if err != nil || len(list) != 1 {
		t.Errorf("ListSettlements: got %+v, %v", list, err)
	}
}

func TestDeleteExpense(t *testing.T) {
	s := newStore(t)
	ctx := t.Context()
	ana := mustUser(t, s, "ana@example.com", "Ana")
	g := seedGroup(t, s, ana)
	other := seedGroup(t, s, ana)

	e, err := s.CreateExpense(ctx, store.NewExpense{
		GroupID: g.ID, PaidBy: ana.ID, CreatedBy: ana.ID, Description: "Museum",
		Amount: 1200, SpentOn: time.Now(), Shares: []ledger.Share{{Member: ana.ID, Amount: 1200}},
	})
	if err != nil {
		t.Fatalf("CreateExpense: %v", err)
	}

	if err := s.DeleteExpense(ctx, other.ID, e.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("delete through another group: got %v, want ErrNotFound", err)
	}
	if err := s.DeleteExpense(ctx, g.ID, e.ID); err != nil {
		t.Errorf("DeleteExpense: %v", err)
	}
	if _, err := s.Expense(ctx, g.ID, e.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("after delete: got %v, want ErrNotFound", err)
	}
}
