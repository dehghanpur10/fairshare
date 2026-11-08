package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/service"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	session, err := s.svc.Register(r.Context(), service.RegisterInput{
		Email: req.Email, Name: req.Name, Password: req.Password,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSession(session))
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	session, err := s.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSession(session))
}

func (s *server) me(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	u, err := s.svc.User(r.Context(), user)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUser(u))
}

func (s *server) listGroups(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groups, err := s.svc.Groups(r.Context(), user)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list[groupResponse]{Items: mapSlice(groups, toGroup)})
}

func (s *server) createGroup(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	var req createGroupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	detail, err := s.svc.CreateGroup(r.Context(), user, req.Name, req.Currency)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", "/v1/groups/"+detail.Group.ID.String())
	writeJSON(w, http.StatusCreated, toGroupDetail(detail))
}

func (s *server) getGroup(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	detail, err := s.svc.Group(r.Context(), user, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toGroupDetail(detail))
}

func (s *server) addMember(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var req addMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	detail, err := s.svc.AddMember(r.Context(), user, groupID, req.Email)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toGroupDetail(detail))
}

func (s *server) listExpenses(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		if limit, err = strconv.Atoi(raw); err != nil {
			s.writeError(w, r, &service.ValidationError{Fields: map[string]string{"limit": "must be a whole number"}})
			return
		}
	}
	page, err := s.svc.Expenses(r.Context(), user, groupID, q.Get("cursor"), limit)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, expensePageResponse{
		Items:      mapSlice(page.Expenses, toExpense),
		NextCursor: page.NextCursor,
	})
}

func (s *server) createExpense(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var req createExpenseRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}

	var spentOn time.Time
	if req.SpentOn != "" {
		if spentOn, err = time.Parse(dateLayout, req.SpentOn); err != nil {
			s.writeError(w, r, &service.ValidationError{Fields: map[string]string{"spent_on": "must be a date like 2026-04-30"}})
			return
		}
	}
	parts := make([]service.SplitPart, len(req.Split.Participants))
	for i, p := range req.Split.Participants {
		parts[i] = service.SplitPart{UserID: p.UserID, Value: p.Value}
	}

	expense, err := s.svc.AddExpense(r.Context(), user, groupID, service.ExpenseInput{
		Description: req.Description,
		Amount:      req.Amount,
		PaidBy:      req.PaidBy,
		SpentOn:     spentOn,
		Method:      service.SplitMethod(req.Split.Method),
		Split:       parts,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toExpense(expense))
}

func (s *server) deleteExpense(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	expenseID, err := pathID(r, "expenseID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if err := s.svc.DeleteExpense(r.Context(), user, groupID, expenseID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) balances(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	summary, err := s.svc.Balances(r.Context(), user, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toBalances(summary))
}

func (s *server) listSettlements(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	settlements, err := s.svc.Settlements(r.Context(), user, groupID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list[settlementResponse]{Items: mapSlice(settlements, toSettlement)})
}

func (s *server) createSettlement(w http.ResponseWriter, r *http.Request, user uuid.UUID) {
	groupID, err := pathID(r, "groupID")
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var req createSettlementRequest
	if err := decodeJSON(w, r, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	settlement, err := s.svc.RecordSettlement(r.Context(), user, groupID, service.SettlementInput{
		From: req.From, To: req.To, Amount: req.Amount,
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSettlement(settlement))
}

// Compile-time check that the store satisfies the readiness dependency.
var _ Pinger = (*store.Store)(nil)
