// Package ledger holds the money math for shared expenses: dividing an amount
// between group members and settling the balances that result.
//
// Amounts are int64 minor units (cents for USD). Nothing in here uses floats.
package ledger

import (
	"cmp"
	"errors"
	"slices"

	"github.com/google/uuid"
)

const (
	// MaxAmount caps a single expense. Keeping it at 10^12 minor units means
	// amount*weight stays well inside int64 during a weighted split.
	MaxAmount int64 = 1_000_000_000_000

	// MaxWeight caps one participant's weight for the same reason.
	MaxWeight int64 = 1_000_000

	// PercentTotal is 100% expressed in basis points.
	PercentTotal int64 = 10_000
)

var (
	ErrNoParticipants  = errors.New("at least one participant is required")
	ErrInvalidAmount   = errors.New("amount is out of range")
	ErrInvalidWeight   = errors.New("weights must be positive")
	ErrInvalidShare    = errors.New("a share is negative or larger than the expense")
	ErrDuplicateMember = errors.New("a participant is listed more than once")
	ErrSharesMismatch  = errors.New("shares do not add up to the expense amount")
	ErrPercentTotal    = errors.New("percentages must add up to 100")
)

// Share is the part of an expense that one member owes.
type Share struct {
	Member uuid.UUID
	Amount int64
}

// Weight is a member's relative claim on an expense.
type Weight struct {
	Member uuid.UUID
	Weight int64
}

// SplitEqual divides total evenly between members. Cents that do not divide
// evenly go one each to the first members in the order given, so the shares
// always sum to total.
func SplitEqual(total int64, members []uuid.UUID) ([]Share, error) {
	weights := make([]Weight, len(members))
	for i, m := range members {
		weights[i] = Weight{Member: m, Weight: 1}
	}
	return SplitWeighted(total, weights)
}

// SplitPercent splits total by percentages given in basis points, so 2500
// means 25%. The percentages must add up to exactly 100%.
func SplitPercent(total int64, weights []Weight) ([]Share, error) {
	var sum int64
	for _, w := range weights {
		if w.Weight <= 0 || w.Weight > PercentTotal {
			return nil, ErrInvalidWeight
		}
		sum += w.Weight
	}
	if sum != PercentTotal {
		return nil, ErrPercentTotal
	}
	return SplitWeighted(total, weights)
}

// SplitWeighted divides total in proportion to each weight using the largest
// remainder method. Everyone first gets the floor of their exact share, then
// the leftover cents go to the largest fractional remainders, with ties going
// to whoever is listed first.
func SplitWeighted(total int64, weights []Weight) ([]Share, error) {
	if err := checkTotal(total); err != nil {
		return nil, err
	}
	if len(weights) == 0 {
		return nil, ErrNoParticipants
	}

	seen := make(map[uuid.UUID]struct{}, len(weights))
	var sum int64
	for _, w := range weights {
		if w.Weight <= 0 || w.Weight > MaxWeight {
			return nil, ErrInvalidWeight
		}
		if _, dup := seen[w.Member]; dup {
			return nil, ErrDuplicateMember
		}
		seen[w.Member] = struct{}{}
		sum += w.Weight
	}

	shares := make([]Share, len(weights))
	remainders := make([]int64, len(weights))
	var allocated int64
	for i, w := range weights {
		exact := total * w.Weight
		shares[i] = Share{Member: w.Member, Amount: exact / sum}
		remainders[i] = exact % sum
		allocated += shares[i].Amount
	}

	order := make([]int, len(weights))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Compare(remainders[b], remainders[a])
	})
	for _, i := range order[:total-allocated] {
		shares[i].Amount++
	}
	return shares, nil
}

// SplitExact checks shares the user entered by hand against total.
func SplitExact(total int64, shares []Share) ([]Share, error) {
	if err := checkTotal(total); err != nil {
		return nil, err
	}
	if len(shares) == 0 {
		return nil, ErrNoParticipants
	}

	seen := make(map[uuid.UUID]struct{}, len(shares))
	var sum int64
	for _, s := range shares {
		if s.Amount < 0 || s.Amount > total {
			return nil, ErrInvalidShare
		}
		if _, dup := seen[s.Member]; dup {
			return nil, ErrDuplicateMember
		}
		seen[s.Member] = struct{}{}
		sum += s.Amount
	}
	if sum != total {
		return nil, ErrSharesMismatch
	}
	return slices.Clone(shares), nil
}

func checkTotal(total int64) error {
	if total <= 0 || total > MaxAmount {
		return ErrInvalidAmount
	}
	return nil
}
