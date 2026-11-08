package ledger

import (
	"cmp"
	"errors"
	"slices"

	"github.com/google/uuid"
)

// ErrUnbalanced means the balances passed to Settle do not sum to zero. That
// points to a bug upstream, not to bad user input.
var ErrUnbalanced = errors.New("balances do not sum to zero")

// Balance is a member's net position in a group. Positive means the group
// owes them money, negative means they owe the group.
type Balance struct {
	Member uuid.UUID
	Net    int64
}

// Transfer is a payment from one member to another.
type Transfer struct {
	From   uuid.UUID
	To     uuid.UUID
	Amount int64
}

// Settle returns transfers that bring every balance to zero.
//
// Debtors and creditors are each sorted largest first and paired off in that
// order. Every transfer clears at least one side, so n members never need
// more than n-1 transfers. Finding the true minimum is NP-hard, so this is
// the usual practical compromise.
func Settle(balances []Balance) ([]Transfer, error) {
	var debtors, creditors []Balance
	var sum int64
	for _, b := range balances {
		sum += b.Net
		switch {
		case b.Net < 0:
			debtors = append(debtors, Balance{Member: b.Member, Net: -b.Net})
		case b.Net > 0:
			creditors = append(creditors, b)
		}
	}
	if sum != 0 {
		return nil, ErrUnbalanced
	}

	largestFirst := func(a, b Balance) int { return cmp.Compare(b.Net, a.Net) }
	slices.SortStableFunc(debtors, largestFirst)
	slices.SortStableFunc(creditors, largestFirst)

	transfers := make([]Transfer, 0, max(len(debtors), len(creditors)))
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		amount := min(debtors[i].Net, creditors[j].Net)
		transfers = append(transfers, Transfer{
			From:   debtors[i].Member,
			To:     creditors[j].Member,
			Amount: amount,
		})
		debtors[i].Net -= amount
		creditors[j].Net -= amount
		if debtors[i].Net == 0 {
			i++
		}
		if creditors[j].Net == 0 {
			j++
		}
	}
	return transfers, nil
}
