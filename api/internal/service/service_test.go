package service

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

func TestBuildShares(t *testing.T) {
	a, b, c := uuid.UUID{15: 1}, uuid.UUID{15: 2}, uuid.UUID{15: 3}
	tests := []struct {
		method SplitMethod
		parts  []SplitPart
		want   []int64
	}{
		{SplitEqual, []SplitPart{{UserID: a}, {UserID: b}, {UserID: c}}, []int64{334, 333, 333}},
		{SplitExact, []SplitPart{{a, 600}, {b, 400}}, []int64{600, 400}},
		{SplitPercent, []SplitPart{{a, 7500}, {b, 2500}}, []int64{750, 250}},
		{SplitShares, []SplitPart{{a, 3}, {b, 1}}, []int64{750, 250}},
	}
	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			shares, err := buildShares(1000, tt.method, tt.parts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := make([]int64, len(shares))
			for i, s := range shares {
				got[i] = s.Amount
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	if _, err := buildShares(1000, SplitPercent, []SplitPart{{a, 5000}}); !errors.Is(err, ledger.ErrPercentTotal) {
		t.Errorf("50%% split: got %v, want ErrPercentTotal", err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	want := &store.Cursor{
		CreatedAt: time.Date(2026, 4, 2, 9, 30, 15, 123456000, time.UTC),
		ID:        uuid.New(),
	}
	got, err := decodeCursor(encodeCursor(want))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) || got.ID != want.ID {
		t.Errorf("got %+v, want %+v", got, want)
	}

	if c, err := decodeCursor(""); c != nil || err != nil {
		t.Errorf("empty cursor: got %+v, %v; want nil, nil", c, err)
	}
	for _, bad := range []string{"%%%", "bm8tc2VwYXJhdG9y", "eHx5"} {
		if _, err := decodeCursor(bad); err == nil {
			t.Errorf("decodeCursor(%q) accepted a malformed cursor", bad)
		}
	}
}

func TestValidEmail(t *testing.T) {
	for email, want := range map[string]bool{
		"ana@example.com":          true,
		"ana.silva+trips@mail.pt":  true,
		"ana":                      false,
		"Ana <ana@example.com>":    false,
		"ana@":                     false,
		"":                         false,
		string(make([]byte, 255)):  false,
		"two@at@signs.example.com": false,
	} {
		if got := validEmail(email); got != want {
			t.Errorf("validEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestValidationErrorListsFieldsInOrder(t *testing.T) {
	err := &ValidationError{Fields: map[string]string{"name": "required", "email": "invalid"}}
	if got, want := err.Error(), "invalid input: email: invalid; name: required"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
