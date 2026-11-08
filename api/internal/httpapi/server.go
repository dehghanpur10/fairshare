// Package httpapi exposes the service layer as a JSON HTTP API.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/auth"
	"github.com/dehghanpur10/fairshare/api/internal/service"
)

// Pinger reports whether a dependency such as the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

type server struct {
	svc            *service.Service
	db             Pinger
	log            *slog.Logger
	allowedOrigins map[string]bool
}

// New returns the API's root handler with all routes and middleware wired.
func New(svc *service.Service, db Pinger, log *slog.Logger, allowedOrigins []string) http.Handler {
	s := &server{svc: svc, db: db, log: log, allowedOrigins: make(map[string]bool)}
	for _, o := range allowedOrigins {
		s.allowedOrigins[o] = true
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)

	mux.HandleFunc("POST /v1/auth/register", s.register)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.HandleFunc("GET /v1/me", s.authed(s.me))

	mux.HandleFunc("GET /v1/groups", s.authed(s.listGroups))
	mux.HandleFunc("POST /v1/groups", s.authed(s.createGroup))
	mux.HandleFunc("GET /v1/groups/{groupID}", s.authed(s.getGroup))
	mux.HandleFunc("POST /v1/groups/{groupID}/members", s.authed(s.addMember))

	mux.HandleFunc("GET /v1/groups/{groupID}/expenses", s.authed(s.listExpenses))
	mux.HandleFunc("POST /v1/groups/{groupID}/expenses", s.authed(s.createExpense))
	mux.HandleFunc("DELETE /v1/groups/{groupID}/expenses/{expenseID}", s.authed(s.deleteExpense))

	mux.HandleFunc("GET /v1/groups/{groupID}/balances", s.authed(s.balances))
	mux.HandleFunc("GET /v1/groups/{groupID}/settlements", s.authed(s.listSettlements))
	mux.HandleFunc("POST /v1/groups/{groupID}/settlements", s.authed(s.createSettlement))

	var h http.Handler = mux
	h = s.cors(h)
	h = secureHeaders(h)
	h = s.recoverPanics(h)
	h = s.logRequests(h)
	h = withRequestID(h)
	return h
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready fails when the database is unreachable so the load balancer stops
// sending traffic to this task until it recovers.
func (s *server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		s.log.WarnContext(ctx, "readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type authedHandler func(w http.ResponseWriter, r *http.Request, user uuid.UUID)

// authed requires a valid bearer token and passes the caller's user ID to h.
func (s *server) authed(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
			s.writeError(w, r, auth.ErrInvalidToken)
			return
		}
		user, err := s.svc.Authenticate(token)
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		h(w, r, user)
	}
}

// pathID parses a UUID path parameter. A malformed ID cannot match anything,
// so it is reported as not found rather than as a bad request.
func pathID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, service.ErrNotFound
	}
	return id, nil
}
