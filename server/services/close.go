package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

func (s *Services) CloseGroup(ctx context.Context, id types.Identity, groupID string) (*types.CloseResult, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("close: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("close: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	status, err := qtx.LockGroup(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("close: lock group", "error", err)
		return nil, types.NewServerError()
	}
	if status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}

	members, err := qtx.ListApprovedMembers(ctx, groupID)
	if err != nil {
		s.logger.Errorw("close: query members", "error", err)
		return nil, types.NewServerError()
	}

	paidRows, err := qtx.SumPaidByMember(ctx, groupID)
	if err != nil {
		s.logger.Errorw("close: query paid", "error", err)
		return nil, types.NewServerError()
	}
	paid := make(map[string]int64, len(paidRows))
	for _, r := range paidRows {
		paid[r.PaidBy] = r.Total
	}

	catRows, err := qtx.SumSpendByCategory(ctx, groupID)
	if err != nil {
		s.logger.Errorw("close: query categories", "error", err)
		return nil, types.NewServerError()
	}
	spendPerCategory := make(map[string]int64, len(catRows))
	for _, r := range catRows {
		spendPerCategory[string(r.Category)] = r.Total
	}

	var total int64
	for _, v := range paid {
		total += v
	}

	shares := fairShares(total, len(members))
	balances := make([]types.MemberBalance, len(members))
	for i, m := range members {
		p := paid[m.ID]
		balances[i] = types.MemberBalance{UserID: m.UserID, Paid: p, FairShare: shares[i], Net: p - shares[i]}
	}
	plan := computePlan(balances)

	snapshot := types.SettlementSnapshot{
		TotalSpent:       total,
		MemberCount:      len(members),
		SpendPerCategory: spendPerCategory,
		Members:          balances,
	}
	snapJSON, err := json.Marshal(snapshot)
	if err != nil {
		s.logger.Errorw("close: marshal snapshot", "error", err)
		return nil, types.NewServerError()
	}

	if err := qtx.MarkGroupClosed(ctx, groupID); err != nil {
		s.logger.Errorw("close: update status", "error", err)
		return nil, types.NewServerError()
	}

	runID, err := qtx.CreateSettlementRun(ctx, repo.CreateSettlementRunParams{
		GroupID:    groupID,
		ComputedBy: caller.UserID,
		Snapshot:   snapJSON,
	})
	if err != nil {
		s.logger.Errorw("close: insert settlement run", "error", err)
		return nil, types.NewServerError()
	}

	for _, t := range plan {
		if err := qtx.CreatePayment(ctx, repo.CreatePaymentParams{
			SettlementRunID: runID,
			GroupID:         groupID,
			FromUserID:      t.From,
			ToUserID:        t.To,
			AmountBaisa:     t.Amount,
		}); err != nil {
			s.logger.Errorw("close: insert payment", "error", err)
			return nil, types.NewServerError()
		}
	}

	// A plan with zero transfers has nothing to confirm: every payment (all
	// none of them) is settled, so the group is fully settled at close.
	if len(plan) == 0 {
		if err := qtx.MarkGroupSettled(ctx, groupID); err != nil {
			s.logger.Errorw("close: mark group settled", "error", err)
			return nil, types.NewServerError()
		}
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     groupID,
		ActorUserID: caller.UserID,
		Action:      "group.closed",
		Before:      map[string]any{"status": types.GroupOpen},
		After:       map[string]any{"status": types.GroupClosed, "total_spent": total, "transfers": len(plan)},
	}); err != nil {
		s.logger.Errorw("close: write audit", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("close: commit", "error", err)
		return nil, types.NewServerError()
	}

	s.logger.Infow("group closed", "group_id", groupID, "total", total, "transfers", len(plan), "by", caller.UserID)
	return &types.CloseResult{GroupID: groupID, Snapshot: snapshot, Plan: plan}, nil
}
