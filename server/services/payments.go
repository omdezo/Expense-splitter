package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) GetSettlementPlan(ctx context.Context, id types.Identity, groupID string) (*types.SettlementPlanResponse, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("settlement plan: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleMember); apiErr != nil {
		return nil, apiErr
	}

	status, err := s.q.GetGroupStatus(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("settlement plan: load status", "error", err)
		return nil, types.NewServerError()
	}
	if status == types.GroupOpen {
		return nil, types.NewConflictError("group is not closed yet — no settlement plan exists")
	}

	rows, err := s.q.ListPayments(ctx, groupID)
	if err != nil {
		s.logger.Errorw("settlement plan: query payments", "error", err)
		return nil, types.NewServerError()
	}

	resp := &types.SettlementPlanResponse{GroupID: groupID, Payments: make([]types.PaymentView, 0, len(rows))}
	for _, r := range rows {
		resp.Payments = append(resp.Payments, types.PaymentView{
			ID:          r.ID,
			From:        r.FromUserID,
			To:          r.ToUserID,
			AmountBaisa: r.AmountBaisa,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
		})
	}
	return resp, nil
}
