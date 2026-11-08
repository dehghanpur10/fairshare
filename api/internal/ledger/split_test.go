package ledger_test

import (
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
)

// ids returns n distinct, readable member IDs: ...0001, ...0002 and so on.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.UUID{15: byte(i + 1)}
	}
	return out
}

func amounts(shares []ledger.Share) []int64 {
	out := make([]int64, len(shares))
	for i, s := range shares {
		out[i] = s.Amount
	}
	return out
}

func weights(members []uuid.UUID, w ...int64) []ledger.Weight {
	out := make([]ledger.Weight, len(w))
	for i := range w {
		out[i] = ledger.Weight{Member: members[i], Weight: w[i]}
	}
	return out
}

func TestSplitEqual(t *testing.T) {
	tests := []struct {
		name    string
		total   int64
		members int
		want    []int64
	}{
		{"divides evenly", 1000, 4, []int64{250, 250, 250, 250}},
		{"leftover cents go to the first members", 1000, 3, []int64{334, 333, 333}},
		{"fewer cents than members", 2, 3, []int64{1, 1, 0}},
		{"single member pays it all", 4999, 1, []int64{4999}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.SplitEqual(tt.total, ids(tt.members))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(amounts(got), tt.want) {
				t.Errorf("got %v, want %v", amounts(got), tt.want)
			}
		})
	}
}

func TestSplitEqualErrors(t *testing.T) {
	m := ids(2)
	tests := []struct {
		name    string
		total   int64
		members []uuid.UUID
		want    error
	}{
		{"zero amount", 0, m, ledger.ErrInvalidAmount},
		{"negative amount", -100, m, ledger.ErrInvalidAmount},
		{"over the cap", ledger.MaxAmount + 1, m, ledger.ErrInvalidAmount},
		{"nobody to split with", 100, nil, ledger.ErrNoParticipants},
		{"same member twice", 100, []uuid.UUID{m[0], m[0]}, ledger.ErrDuplicateMember},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ledger.SplitEqual(tt.total, tt.members)
			if !errors.Is(err, tt.want) {
				t.Errorf("got error %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSplitWeighted(t *testing.T) {
	m := ids(3)
	tests := []struct {
		name    string
		total   int64
		weights []ledger.Weight
		want    []int64
	}{
		{"two shares to one", 1000, weights(m, 2, 1, 1), []int64{500, 250, 250}},
		{"largest remainder wins the extra cent", 10, weights(m, 1, 2), []int64{3, 7}},
		{"ties go to whoever is listed first", 100, weights(m, 1, 1, 1), []int64{34, 33, 33}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ledger.SplitWeighted(tt.total, tt.weights)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(amounts(got), tt.want) {
				t.Errorf("got %v, want %v", amounts(got), tt.want)
			}
		})
	}

	if _, err := ledger.SplitWeighted(100, weights(m, 1, 0)); !errors.Is(err, ledger.ErrInvalidWeight) {
		t.Errorf("zero weight: got %v, want ErrInvalidWeight", err)
	}
}

func TestSplitPercent(t *testing.T) {
	m := ids(3)

	got, err := ledger.SplitPercent(10_000, weights(m, 5000, 3000, 2000))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int64{5000, 3000, 2000}; !slices.Equal(amounts(got), want) {
		t.Errorf("got %v, want %v", amounts(got), want)
	}

	got, err = ledger.SplitPercent(333, weights(m, 3333, 3333, 3334))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int64{111, 111, 111}; !slices.Equal(amounts(got), want) {
		t.Errorf("thirds of 333: got %v, want %v", amounts(got), want)
	}

	if _, err := ledger.SplitPercent(100, weights(m, 5000, 4000)); !errors.Is(err, ledger.ErrPercentTotal) {
		t.Errorf("90%%: got %v, want ErrPercentTotal", err)
	}
}

func TestSplitExact(t *testing.T) {
	m := ids(3)
	share := func(i int, amount int64) ledger.Share { return ledger.Share{Member: m[i], Amount: amount} }

	got, err := ledger.SplitExact(1000, []ledger.Share{share(0, 700), share(1, 300), share(2, 0)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []int64{700, 300, 0}; !slices.Equal(amounts(got), want) {
		t.Errorf("got %v, want %v", amounts(got), want)
	}

	tests := []struct {
		name   string
		shares []ledger.Share
		want   error
	}{
		{"short by a cent", []ledger.Share{share(0, 500), share(1, 499)}, ledger.ErrSharesMismatch},
		{"negative share", []ledger.Share{share(0, 1100), share(1, -100)}, ledger.ErrInvalidShare},
		{"same member twice", []ledger.Share{share(0, 500), share(0, 500)}, ledger.ErrDuplicateMember},
		{"no shares", nil, ledger.ErrNoParticipants},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ledger.SplitExact(1000, tt.shares); !errors.Is(err, tt.want) {
				t.Errorf("got error %v, want %v", err, tt.want)
			}
		})
	}
}

// Every weighted split must account for each cent and give each member
// either the floor or the ceiling of their exact proportional share.
func TestSplitWeightedProperties(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 11))
	for range 2000 {
		n := 1 + rng.IntN(15)
		total := 1 + rng.Int64N(10_000_000)
		ws := make([]ledger.Weight, n)
		var weightSum int64
		for i, id := range ids(n) {
			ws[i] = ledger.Weight{Member: id, Weight: 1 + rng.Int64N(10_000)}
			weightSum += ws[i].Weight
		}

		shares, err := ledger.SplitWeighted(total, ws)
		if err != nil {
			t.Fatalf("SplitWeighted(%d, %v): %v", total, ws, err)
		}

		var sum int64
		for i, s := range shares {
			sum += s.Amount
			floor := total * ws[i].Weight / weightSum
			if s.Amount != floor && s.Amount != floor+1 {
				t.Fatalf("share %d of %d is %d, want %d or %d", i, total, s.Amount, floor, floor+1)
			}
		}
		if sum != total {
			t.Fatalf("shares sum to %d, want %d", sum, total)
		}
	}
}
