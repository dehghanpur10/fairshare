package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/dehghanpur10/fairshare/api/internal/ledger"
	"github.com/dehghanpur10/fairshare/api/internal/store"
)

type SettlementInput struct {
	From   uuid.UUID
	To     uuid.UUID
	Amount int64
}

// BalanceSummary is each member's net position plus the transfers that
// would settle the group.
type BalanceSummary struct {
	Balances  []store.MemberBalance
	Transfers []ledger.Transfer
}

func (s *Service) Balances(ctx context.Context, actor, groupID uuid.UUID) (BalanceSummary, error) {
	if _, err := s.requireMember(ctx, groupID, actor); err != nil {
		return BalanceSummary{}, err
	}
	balances, err := s.store.Balances(ctx, groupID)
	if err != nil {
		return BalanceSummary{}, err
	}
	nets := make([]ledger.Balance, len(balances))
	for i, b := range balances {
		nets[i] = ledger.Balance{Member: b.UserID, Net: b.Net}
	}
	transfers, err := ledger.Settle(nets)
	if err != nil {
		return BalanceSummary{}, fmt.Errorf("settle group %s: %w", groupID, err)
	}
	return BalanceSummary{Balances: balances, Transfers: transfers}, nil
}

// RecordSettlement records that one member paid another back. Either person
// involved may record it, and so may the group owner.
func (s *Service) RecordSettlement(ctx context.Context, actor, groupID uuid.UUID, in SettlementInput) (store.Settlement, error) {
	role, err := s.requireMember(ctx, groupID, actor)
	if err != nil {
		return store.Settlement{}, err
	}

	v := validator{}
	v.check(in.Amount > 0 && in.Amount <= ledger.MaxAmount, "amount", "must be a positive amount in minor units")
	v.check(in.From != in.To, "to", "must be someone other than the payer")
	if err := v.err(); err != nil {
		return store.Settlement{}, err
	}
	if actor != in.From && actor != in.To && role != store.RoleOwner {
		return store.Settlement{}, ErrForbidden
	}

	st, err := s.store.CreateSettlement(ctx, store.NewSettlement{
		GroupID:   groupID,
		From:      in.From,
		To:        in.To,
		Amount:    in.Amount,
		CreatedBy: actor,
	})
	if errors.Is(err, store.ErrNotMember) {
		return store.Settlement{}, &ValidationError{Fields: map[string]string{"to": "both people must be members of the group"}}
	}
	return st, err
}

func (s *Service) Settlements(ctx context.Context, actor, groupID uuid.UUID) ([]store.Settlement, error) {
	if _, err := s.requireMember(ctx, groupID, actor); err != nil {
		return nil, err
	}
	return s.store.ListSettlements(ctx, groupID)
}
