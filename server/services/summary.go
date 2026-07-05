package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

// GroupSummary computes the financial overview live from current expenses. For
// closed groups the expenses are frozen, so this matches the settlement
// snapshot; the math is the same fairShares used at close.
func (s *Services) GroupSummary(ctx context.Context, id types.Identity, groupID string) (*types.GroupSummary, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("summary: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleMember); apiErr != nil {
		return nil, apiErr
	}

	g, err := s.q.GetGroupByID(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("summary: load group", "error", err)
		return nil, types.NewServerError()
	}

	members, err := s.q.ListApprovedMembers(ctx, groupID)
	if err != nil {
		s.logger.Errorw("summary: query members", "error", err)
		return nil, types.NewServerError()
	}

	paidRows, err := s.q.SumPaidByMember(ctx, groupID)
	if err != nil {
		s.logger.Errorw("summary: query paid", "error", err)
		return nil, types.NewServerError()
	}
	paid := make(map[string]int64, len(paidRows))
	var total int64
	for _, r := range paidRows {
		paid[r.PaidBy] = r.Total
		total += r.Total
	}

	catRows, err := s.q.SumSpendByCategory(ctx, groupID)
	if err != nil {
		s.logger.Errorw("summary: query categories", "error", err)
		return nil, types.NewServerError()
	}
	spendPerCategory := make(map[string]int64, len(catRows))
	for _, r := range catRows {
		spendPerCategory[string(r.Category)] = r.Total
	}

	shares := fairShares(total, len(members))
	summary := &types.GroupSummary{
		GroupID:          g.ID,
		Name:             g.Name,
		StartDate:        g.StartDate,
		EndDate:          g.EndDate,
		Status:           g.Status,
		MemberCount:      len(members),
		TotalSpent:       total,
		SpendPerCategory: spendPerCategory,
		Members:          make([]types.MemberSummary, 0, len(members)),
	}
	for i, m := range members {
		p := paid[m.ID]
		summary.Members = append(summary.Members, types.MemberSummary{
			UserID:    m.UserID,
			Email:     m.Email,
			Paid:      p,
			FairShare: shares[i],
			Net:       p - shares[i],
		})
	}
	return summary, nil
}

// PublicGroupStatus is the only unauthenticated read: shareable status by token,
// no per-person detail.
func (s *Services) PublicGroupStatus(ctx context.Context, statusToken string) (*types.PublicGroupStatus, types.APIError) {
	row, err := s.q.GetGroupPublicStatus(ctx, statusToken)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("unknown group token")
	case err != nil:
		s.logger.Errorw("public status: query", "error", err)
		return nil, types.NewServerError()
	}
	return &types.PublicGroupStatus{
		Name:        row.Name,
		Status:      row.Status,
		TotalSpent:  row.TotalSpent,
		MemberCount: int(row.MemberCount),
	}, nil
}
