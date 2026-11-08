package ledger_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
)

func TestSettle(t *testing.T) {
	m := ids(4)
	tests := []struct {
		name     string
		balances []ledger.Balance
		want     []ledger.Transfer
	}{
		{
			name:     "already settled",
			balances: []ledger.Balance{{Member: m[0]}, {Member: m[1]}},
			want:     []ledger.Transfer{},
		},
		{
			name:     "one debt",
			balances: []ledger.Balance{{Member: m[0], Net: 500}, {Member: m[1], Net: -500}},
			want:     []ledger.Transfer{{From: m[1], To: m[0], Amount: 500}},
		},
		{
			name: "one payer covered everyone",
			balances: []ledger.Balance{
				{Member: m[0], Net: 900},
				{Member: m[1], Net: -300},
				{Member: m[2], Net: -300},
				{Member: m[3], Net: -300},
			},
			want: []ledger.Transfer{
				{From: m[1], To: m[0], Amount: 300},
				{From: m[2], To: m[0], Amount: 300},
				{From: m[3], To: m[0], Amount: 300},
			},
		},
		{
			name: "largest debtor pays largest creditor first",
			balances: []ledger.Balance{
				{Member: m[0], Net: 200},
				{Member: m[1], Net: -700},
				{Member: m[2], Net: 600},
				{Member: m[3], Net: -100},
			},
			want: []ledger.Transfer{
				{From: m[1], To: m[2], Amount: 600},
				{From: m[1], To: m[0], Amount: 100},
				{From: m[3], To: m[0], Amount: 100},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.Settle(tt.balances)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSettleRejectsUnbalancedInput(t *testing.T) {
	m := ids(2)
	_, err := ledger.Settle([]ledger.Balance{{Member: m[0], Net: 100}, {Member: m[1], Net: -99}})
	if !errors.Is(err, ledger.ErrUnbalanced) {
		t.Errorf("got %v, want ErrUnbalanced", err)
	}
}

// Applying the transfers must zero every balance, using at most n-1 strictly
// positive payments and never asking someone to pay themselves.
func TestSettleProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(3, 5))
	for range 2000 {
		n := 2 + rng.IntN(12)
		members := ids(n)
		balances := make([]ledger.Balance, n)
		var running int64
		for i := range n - 1 {
			net := rng.Int64N(200_001) - 100_000
			balances[i] = ledger.Balance{Member: members[i], Net: net}
			running += net
		}
		balances[n-1] = ledger.Balance{Member: members[n-1], Net: -running}

		transfers, err := ledger.Settle(balances)
		if err != nil {
			t.Fatalf("Settle(%v): %v", balances, err)
		}
		if len(transfers) > n-1 {
			t.Fatalf("%d members needed %d transfers", n, len(transfers))
		}

		net := make(map[uuid.UUID]int64, n)
		for _, b := range balances {
			net[b.Member] = b.Net
		}
		for _, tr := range transfers {
			if tr.Amount <= 0 {
				t.Fatalf("non-positive transfer %v", tr)
			}
			if tr.From == tr.To {
				t.Fatalf("self transfer %v", tr)
			}
			net[tr.From] += tr.Amount
			net[tr.To] -= tr.Amount
		}
		for id, v := range net {
			if v != 0 {
				t.Fatalf("member %x left with %d after settling %v", id, v, balances)
			}
		}
	}
}
