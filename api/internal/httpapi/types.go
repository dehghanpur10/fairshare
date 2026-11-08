package httpapi

import (
	"time"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
	"github.com/dehghanpur10/fairshare/api/internal/service"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

// The wire format lives here, separate from store and service types, so
// renaming a column or a Go field never silently changes the public API.
// Field names match api/openapi.yaml.

const dateLayout = time.DateOnly

type registerRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createGroupRequest struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type addMemberRequest struct {
	Email string `json:"email"`
}

type createExpenseRequest struct {
	Description string    `json:"description"`
	Amount      int64     `json:"amount"`
	PaidBy      uuid.UUID `json:"paid_by"`
	SpentOn     string    `json:"spent_on"`
	Split       struct {
		Method       string `json:"method"`
		Participants []struct {
			UserID uuid.UUID `json:"user_id"`
			Value  int64     `json:"value"`
		} `json:"participants"`
	} `json:"split"`
}

type createSettlementRequest struct {
	From   uuid.UUID `json:"from"`
	To     uuid.UUID `json:"to"`
	Amount int64     `json:"amount"`
}

type list[T any] struct {
	Items []T `json:"items"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type sessionResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        userResponse `json:"user"`
}

type groupResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Currency  string    `json:"currency"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type memberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}

type groupDetailResponse struct {
	groupResponse
	Members []memberResponse `json:"members"`
}

type shareResponse struct {
	UserID uuid.UUID `json:"user_id"`
	Amount int64     `json:"amount"`
}

type expenseResponse struct {
	ID          uuid.UUID       `json:"id"`
	GroupID     uuid.UUID       `json:"group_id"`
	PaidBy      uuid.UUID       `json:"paid_by"`
	Description string          `json:"description"`
	Amount      int64           `json:"amount"`
	SpentOn     string          `json:"spent_on"`
	CreatedBy   uuid.UUID       `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	Shares      []shareResponse `json:"shares"`
}

type expensePageResponse struct {
	Items      []expenseResponse `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type balanceResponse struct {
	UserID uuid.UUID `json:"user_id"`
	Name   string    `json:"name"`
	Net    int64     `json:"net"`
}

type transferResponse struct {
	From   uuid.UUID `json:"from"`
	To     uuid.UUID `json:"to"`
	Amount int64     `json:"amount"`
}

type balancesResponse struct {
	Balances  []balanceResponse  `json:"balances"`
	Transfers []transferResponse `json:"transfers"`
}

type settlementResponse struct {
	ID        uuid.UUID `json:"id"`
	GroupID   uuid.UUID `json:"group_id"`
	From      uuid.UUID `json:"from"`
	To        uuid.UUID `json:"to"`
	Amount    int64     `json:"amount"`
	CreatedBy uuid.UUID `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// mapSlice converts a slice and never returns nil, so empty lists encode as
// [] rather than null.
func mapSlice[T, R any](in []T, f func(T) R) []R {
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

func toUser(u store.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt}
}

func toSession(s service.Session) sessionResponse {
	return sessionResponse{AccessToken: s.Token, TokenType: "Bearer", ExpiresAt: s.ExpiresAt, User: toUser(s.User)}
}

func toGroup(g store.Group) groupResponse {
	return groupResponse{ID: g.ID, Name: g.Name, Currency: g.Currency, CreatedBy: g.CreatedBy, CreatedAt: g.CreatedAt}
}

func toGroupDetail(d service.GroupDetail) groupDetailResponse {
	return groupDetailResponse{
		groupResponse: toGroup(d.Group),
		Members: mapSlice(d.Members, func(m store.Member) memberResponse {
			return memberResponse{UserID: m.UserID, Name: m.Name, Email: m.Email, Role: m.Role, JoinedAt: m.JoinedAt}
		}),
	}
}

func toExpense(e store.Expense) expenseResponse {
	return expenseResponse{
		ID:          e.ID,
		GroupID:     e.GroupID,
		PaidBy:      e.PaidBy,
		Description: e.Description,
		Amount:      e.Amount,
		SpentOn:     e.SpentOn.Format(dateLayout),
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		Shares: mapSlice(e.Shares, func(s ledger.Share) shareResponse {
			return shareResponse{UserID: s.Member, Amount: s.Amount}
		}),
	}
}

func toBalances(b service.BalanceSummary) balancesResponse {
	return balancesResponse{
		Balances: mapSlice(b.Balances, func(m store.MemberBalance) balanceResponse {
			return balanceResponse{UserID: m.UserID, Name: m.Name, Net: m.Net}
		}),
		Transfers: mapSlice(b.Transfers, func(t ledger.Transfer) transferResponse {
			return transferResponse{From: t.From, To: t.To, Amount: t.Amount}
		}),
	}
}

func toSettlement(s store.Settlement) settlementResponse {
	return settlementResponse{
		ID: s.ID, GroupID: s.GroupID, From: s.From, To: s.To,
		Amount: s.Amount, CreatedBy: s.CreatedBy, CreatedAt: s.CreatedAt,
	}
}
