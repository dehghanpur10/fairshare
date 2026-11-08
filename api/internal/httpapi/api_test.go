package httpapi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/httpapi"
	"github.com/dehghanpur10/fairshare/api/internal/service"
	"github.com/dehghanpur10/fairshare/api/internal/store"
	"github.com/dehghanpur10/fairshare/api/internal/testdb"
)

const allowedOrigin = "http://localhost:5173"

type api struct {
	t   *testing.T
	url string
}

func newAPI(t *testing.T) *api {
	t.Parallel()
	st := store.New(testdb.New(t))
	tokens := auth.NewTokens([]byte("test-secret-test-secret-test-secret!"), time.Hour)
	svc, err := service.New(st, auth.NewHasher(bcrypt.MinCost), tokens)
	if err != nil {
		t.Fatalf("service.New: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httptest.NewServer(httpapi.New(svc, st, logger, []string{allowedOrigin}))
	t.Cleanup(srv.Close)
	return &api{t: t, url: srv.URL}
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (a *api) do(method, path, token string, body any) response {
	a.t.Helper()
	var reader io.Reader
	if body != nil {
		if raw, ok := body.(string); ok {
			reader = strings.NewReader(raw)
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				a.t.Fatalf("marshal: %v", err)
			}
			reader = bytes.NewReader(b)
		}
	}
	req, err := http.NewRequestWithContext(a.t.Context(), method, a.url+path, reader)
	if err != nil {
		a.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		a.t.Fatalf("read body: %v", err)
	}
	return response{status: res.StatusCode, header: res.Header, body: b}
}

func (a *api) expect(want int, method, path, token string, body any) response {
	a.t.Helper()
	res := a.do(method, path, token, body)
	if res.status != want {
		a.t.Fatalf("%s %s: status %d, want %d; body %s", method, path, res.status, want, res.body)
	}
	return res
}

type user struct {
	ID    string
	Token string
}

func (a *api) register(email, name string) user {
	a.t.Helper()
	var s struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	a.expect(http.StatusCreated, "POST", "/v1/auth/register", "",
		map[string]string{"email": email, "name": name, "password": "s3cret-pass"}).decode(a.t, &s)
	return user{ID: s.User.ID, Token: s.AccessToken}
}

type problem struct {
	Status    int               `json:"status"`
	Detail    string            `json:"detail"`
	Errors    map[string]string `json:"errors"`
	RequestID string            `json:"request_id"`
}

func TestHealth(t *testing.T) {
	a := newAPI(t)
	a.expect(http.StatusOK, "GET", "/healthz", "", nil)
	res := a.expect(http.StatusOK, "GET", "/readyz", "", nil)
	if res.header.Get("X-Request-ID") == "" {
		t.Error("response has no X-Request-ID")
	}
	if res.header.Get("Cache-Control") != "no-store" {
		t.Error("API responses must not be cacheable")
	}
}

func TestAuth(t *testing.T) {
	a := newAPI(t)
	ana := a.register("ana@example.com", "Ana")

	var p problem
	a.expect(http.StatusConflict, "POST", "/v1/auth/register", "",
		map[string]string{"email": "ANA@example.com", "name": "Imposter", "password": "whatever-123"})

	a.expect(http.StatusUnprocessableEntity, "POST", "/v1/auth/register", "",
		map[string]string{"email": "not-an-email", "name": "", "password": "short"}).decode(t, &p)
	for _, field := range []string{"email", "name", "password"} {
		if p.Errors[field] == "" {
			t.Errorf("validation response missing %q: %+v", field, p.Errors)
		}
	}

	res := a.expect(http.StatusUnauthorized, "POST", "/v1/auth/login", "",
		map[string]string{"email": "ana@example.com", "password": "wrong-password"})
	if !strings.HasPrefix(res.header.Get("Content-Type"), "application/problem+json") {
		t.Errorf("error content type %q", res.header.Get("Content-Type"))
	}
	a.expect(http.StatusUnauthorized, "POST", "/v1/auth/login", "",
		map[string]string{"email": "nobody@example.com", "password": "s3cret-pass"})
	a.expect(http.StatusOK, "POST", "/v1/auth/login", "",
		map[string]string{"email": "Ana@Example.com", "password": "s3cret-pass"})

	var me struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	a.expect(http.StatusOK, "GET", "/v1/me", ana.Token, nil).decode(t, &me)
	if me.ID != ana.ID || me.Email != "ana@example.com" {
		t.Errorf("GET /v1/me returned %+v", me)
	}
	a.expect(http.StatusUnauthorized, "GET", "/v1/me", "", nil)
	a.expect(http.StatusUnauthorized, "GET", "/v1/me", "not-a-token", nil)
}

func TestSplittingAndSettlingUp(t *testing.T) {
	a := newAPI(t)
	ana := a.register("ana@example.com", "Ana")
	ben := a.register("ben@example.com", "Ben")
	cat := a.register("cat@example.com", "Cat")

	var group struct {
		ID      string `json:"id"`
		Members []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	res := a.expect(http.StatusCreated, "POST", "/v1/groups", ana.Token,
		map[string]string{"name": "Lisbon trip", "currency": "eur"})
	res.decode(t, &group)
	if res.header.Get("Location") != "/v1/groups/"+group.ID {
		t.Errorf("Location header %q", res.header.Get("Location"))
	}
	groupPath := "/v1/groups/" + group.ID

	a.expect(http.StatusOK, "POST", groupPath+"/members", ana.Token, map[string]string{"email": "ben@example.com"})
	a.expect(http.StatusConflict, "POST", groupPath+"/members", ana.Token, map[string]string{"email": "ben@example.com"})
	a.expect(http.StatusForbidden, "POST", groupPath+"/members", ben.Token, map[string]string{"email": "cat@example.com"})
	a.expect(http.StatusNotFound, "GET", groupPath, cat.Token, nil)
	a.expect(http.StatusNotFound, "GET", "/v1/groups/not-a-uuid", ana.Token, nil)

	// Ana pays 120.00 for dinner, split evenly between her and Ben.
	var expense struct {
		ID     string `json:"id"`
		Shares []struct {
			Amount int64 `json:"amount"`
		} `json:"shares"`
	}
	a.expect(http.StatusCreated, "POST", groupPath+"/expenses", ana.Token, map[string]any{
		"description": "Dinner at Taberna",
		"amount":      12000,
		"spent_on":    "2026-05-14",
		"split": map[string]any{
			"method":       "equal",
			"participants": []map[string]string{{"user_id": ana.ID}, {"user_id": ben.ID}},
		},
	}).decode(t, &expense)
	if len(expense.Shares) != 2 || expense.Shares[0].Amount != 6000 {
		t.Fatalf("unexpected shares %+v", expense.Shares)
	}

	var p problem
	a.expect(http.StatusUnprocessableEntity, "POST", groupPath+"/expenses", ana.Token, map[string]any{
		"description": "Tram", "amount": 300,
		"split": map[string]any{
			"method":       "percent",
			"participants": []map[string]any{{"user_id": ana.ID, "value": 5000}, {"user_id": cat.ID, "value": 5000}},
		},
	}).decode(t, &p)
	if p.Errors["split"] == "" {
		t.Errorf("splitting with an outsider: errors %+v", p.Errors)
	}

	var balances struct {
		Balances []struct {
			UserID string `json:"user_id"`
			Net    int64  `json:"net"`
		} `json:"balances"`
		Transfers []struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Amount int64  `json:"amount"`
		} `json:"transfers"`
	}
	a.expect(http.StatusOK, "GET", groupPath+"/balances", ben.Token, nil).decode(t, &balances)
	if len(balances.Transfers) != 1 || balances.Transfers[0].From != ben.ID ||
		balances.Transfers[0].To != ana.ID || balances.Transfers[0].Amount != 6000 {
		t.Fatalf("expected Ben to owe Ana 60.00, got %+v", balances)
	}

	a.expect(http.StatusForbidden, "DELETE", groupPath+"/expenses/"+expense.ID, ben.Token, nil)
	a.expect(http.StatusCreated, "POST", groupPath+"/settlements", ben.Token,
		map[string]any{"from": ben.ID, "to": ana.ID, "amount": 6000})

	a.expect(http.StatusOK, "GET", groupPath+"/balances", ana.Token, nil).decode(t, &balances)
	if len(balances.Transfers) != 0 {
		t.Errorf("group should be settled, got transfers %+v", balances.Transfers)
	}
	for _, b := range balances.Balances {
		if b.Net != 0 {
			t.Errorf("member %s still has net %d", b.UserID, b.Net)
		}
	}

	var settlements struct {
		Items []json.RawMessage `json:"items"`
	}
	a.expect(http.StatusOK, "GET", groupPath+"/settlements", ana.Token, nil).decode(t, &settlements)
	if len(settlements.Items) != 1 {
		t.Errorf("got %d settlements, want 1", len(settlements.Items))
	}

	a.expect(http.StatusNoContent, "DELETE", groupPath+"/expenses/"+expense.ID, ana.Token, nil)
	a.expect(http.StatusNotFound, "DELETE", groupPath+"/expenses/"+expense.ID, ana.Token, nil)
}

func TestExpensePages(t *testing.T) {
	a := newAPI(t)
	ana := a.register("ana@example.com", "Ana")
	var group struct {
		ID string `json:"id"`
	}
	a.expect(http.StatusCreated, "POST", "/v1/groups", ana.Token,
		map[string]string{"name": "Flat", "currency": "GBP"}).decode(t, &group)
	path := "/v1/groups/" + group.ID + "/expenses"

	for _, d := range []string{"Rent", "Internet", "Cleaning"} {
		a.expect(http.StatusCreated, "POST", path, ana.Token, map[string]any{
			"description": d, "amount": 1000,
			"split": map[string]any{"method": "equal", "participants": []map[string]string{{"user_id": ana.ID}}},
		})
	}

	type page struct {
		Items      []struct{ Description string } `json:"items"`
		NextCursor string                         `json:"next_cursor"`
	}
	var first, second page
	a.expect(http.StatusOK, "GET", path+"?limit=2", ana.Token, nil).decode(t, &first)
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("first page: %+v", first)
	}
	a.expect(http.StatusOK, "GET", path+"?limit=2&cursor="+first.NextCursor, ana.Token, nil).decode(t, &second)
	if len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("second page: %+v", second)
	}

	a.expect(http.StatusUnprocessableEntity, "GET", path+"?limit=500", ana.Token, nil)
	a.expect(http.StatusUnprocessableEntity, "GET", path+"?cursor=garbage", ana.Token, nil)
}

func TestMalformedRequests(t *testing.T) {
	a := newAPI(t)
	tests := []struct {
		name   string
		body   string
		ctype  string
		status int
	}{
		{"broken JSON", `{"email":`, "application/json", http.StatusBadRequest},
		{"unknown field", `{"email":"a@b.co","name":"A","password":"12345678","admin":true}`, "application/json", http.StatusBadRequest},
		{"wrong type", `{"email":42}`, "application/json", http.StatusBadRequest},
		{"two objects", `{"email":"a@b.co"}{"email":"c@d.co"}`, "application/json", http.StatusBadRequest},
		{"empty body", ``, "application/json", http.StatusBadRequest},
		{"form post", `email=a@b.co`, "application/x-www-form-urlencoded", http.StatusUnsupportedMediaType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, a.url+"/v1/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.ctype)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tt.status {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("status %d, want %d; body %s", res.StatusCode, tt.status, body)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	a := newAPI(t)
	preflight := func(origin string) (int, http.Header) {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodOptions, a.url+"/v1/groups", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", http.MethodPost)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		return res.StatusCode, res.Header
	}

	status, header := preflight(allowedOrigin)
	if status != http.StatusNoContent || header.Get("Access-Control-Allow-Origin") != allowedOrigin {
		t.Errorf("allowed origin: status %d, ACAO %q", status, header.Get("Access-Control-Allow-Origin"))
	}
	if _, header := preflight("https://evil.example"); header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers sent to an origin that is not allowed")
	}
}
