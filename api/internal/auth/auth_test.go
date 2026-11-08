package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestHasher(t *testing.T) {
	h := NewHasher(bcrypt.MinCost)

	hash, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("hash is the plaintext password")
	}
	if !h.Check(hash, "correct horse battery staple") {
		t.Error("Check rejected the right password")
	}
	if h.Check(hash, "Correct horse battery staple") {
		t.Error("Check accepted the wrong password")
	}
}

var secret = []byte("0123456789abcdef0123456789abcdef")

func tokensAt(now time.Time) *Tokens {
	tk := NewTokens(secret, time.Hour)
	tk.now = func() time.Time { return now }
	return tk
}

func TestTokenRoundTrip(t *testing.T) {
	issuedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()

	raw, expiresAt, err := tokensAt(issuedAt).Issue(userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if want := issuedAt.Add(time.Hour); !expiresAt.Equal(want) {
		t.Errorf("expires at %v, want %v", expiresAt, want)
	}

	got, err := tokensAt(issuedAt.Add(59 * time.Minute)).Verify(raw)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got != userID {
		t.Errorf("got user %v, want %v", got, userID)
	}
}

func TestTokenRejected(t *testing.T) {
	issuedAt := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	valid, _, err := tokensAt(issuedAt).Issue(userID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	sign := func(method jwt.SigningMethod, key any, claims jwt.RegisteredClaims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		return s
	}
	baseClaims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(time.Hour)),
	}
	withIssuer := baseClaims
	withIssuer.Issuer = "someone-else"
	noExpiry := baseClaims
	noExpiry.ExpiresAt = nil

	tests := []struct {
		name  string
		token string
		at    time.Duration
	}{
		{"expired", valid, 61 * time.Minute},
		{"wrong secret", sign(jwt.SigningMethodHS256, []byte("another-secret-another-secret-00"), baseClaims), 0},
		{"different algorithm", sign(jwt.SigningMethodHS512, secret, baseClaims), 0},
		{"unsigned", sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, baseClaims), 0},
		{"wrong issuer", sign(jwt.SigningMethodHS256, secret, withIssuer), 0},
		{"no expiry", sign(jwt.SigningMethodHS256, secret, noExpiry), 0},
		{"garbage", "not.a.token", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tokensAt(issuedAt.Add(tt.at)).Verify(tt.token); !errors.Is(err, ErrInvalidToken) {
				t.Errorf("got %v, want ErrInvalidToken", err)
			}
		})
	}
}
