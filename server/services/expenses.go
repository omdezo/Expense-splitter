package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) RecordExpense(ctx context.Context, id types.Identity, groupID string, req types.RecordExpenseRequest) (*types.Expense, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("record expense: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("record expense: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)

	
	var status types.GroupStatus
	var inRange bool
	err = tx.QueryRow(ctx,
		`SELECT status, ($1::date BETWEEN start_date::date AND end_date::date) FROM groups WHERE id = $2::uuid FOR SHARE`,
		req.OccurredOn, groupID).Scan(&status, &inRange)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("record expense: load group", "error", err)
		return nil, types.NewServerError()
	}
	if status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}
	if !inRange {
		return nil, types.NewBadRequestError("occurred_on is outside the trip dates")
	}

	// The caller must be an approved member; their membership id is paid_by, so
	// paid_by == caller by construction (no way to record for someone else).
	var membershipID string
	var mStatus types.MembershipStatus
	err = tx.QueryRow(ctx, `SELECT id, status FROM memberships WHERE group_id = $1::uuid AND user_id = $2::uuid`, groupID, caller.UserID).Scan(&membershipID, &mStatus)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("you are not a member of this group")
	case err != nil:
		s.logger.Errorw("record expense: load membership", "error", err)
		return nil, types.NewServerError()
	}
	if mStatus != types.MembershipApproved {
		return nil, types.NewForbiddenError("your membership is not approved")
	}

	e := &types.Expense{
		GroupID:     groupID,
		PaidBy:      caller.UserID,
		AmountBaisa: req.AmountBaisa,
		Category:    req.Category,
		Description: req.Description,
		OccurredOn:  req.OccurredOn,
	}
	const ins = `
INSERT INTO expenses (group_id, paid_by, amount_baisa, category, description, occurred_on)
VALUES ($1::uuid, $2::uuid, $3, $4::expense_category, $5, $6::date)
RETURNING id, created_at`
	if err := tx.QueryRow(ctx, ins, groupID, membershipID, req.AmountBaisa, string(req.Category), req.Description, req.OccurredOn).
		Scan(&e.ID, &e.CreatedAt); err != nil {
		s.logger.Errorw("record expense: insert", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("record expense: commit", "error", err)
		return nil, types.NewServerError()
	}

	return e, nil
}
