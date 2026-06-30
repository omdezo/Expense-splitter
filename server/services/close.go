package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

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

	// Lock the group row and verify it is open — guarantees close + settlement
	// happen exactly once over a consistent snapshot.
	var status types.GroupStatus
	err = tx.QueryRow(ctx, `SELECT status FROM groups WHERE id = $1::uuid FOR UPDATE`, groupID).Scan(&status)
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

	// Approved members, ordered by user_id so the remainder distribution is stable.
	type member struct{ membershipID, userID string }
	var members []member
	rows, err := tx.Query(ctx, `SELECT id, user_id FROM memberships WHERE group_id = $1::uuid AND status = 'approved' ORDER BY user_id`, groupID)
	if err != nil {
		s.logger.Errorw("close: query members", "error", err)
		return nil, types.NewServerError()
	}
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.membershipID, &m.userID); err != nil {
			rows.Close()
			s.logger.Errorw("close: scan member", "error", err)
			return nil, types.NewServerError()
		}
		members = append(members, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.logger.Errorw("close: members rows", "error", err)
		return nil, types.NewServerError()
	}

	// Amount each member paid (expenses.paid_by is the membership id).
	paid := map[string]int64{}
	prows, err := tx.Query(ctx, `SELECT paid_by, SUM(amount_baisa) FROM expenses WHERE group_id = $1::uuid AND deleted_at IS NULL GROUP BY paid_by`, groupID)
	if err != nil {
		s.logger.Errorw("close: query paid", "error", err)
		return nil, types.NewServerError()
	}
	for prows.Next() {
		var membershipID string
		var sum int64
		if err := prows.Scan(&membershipID, &sum); err != nil {
			prows.Close()
			s.logger.Errorw("close: scan paid", "error", err)
			return nil, types.NewServerError()
		}
		paid[membershipID] = sum
	}
	prows.Close()
	if err := prows.Err(); err != nil {
		s.logger.Errorw("close: paid rows", "error", err)
		return nil, types.NewServerError()
	}

	// Spend per category for the snapshot.
	spendPerCategory := map[string]int64{}
	crows, err := tx.Query(ctx, `SELECT category, SUM(amount_baisa) FROM expenses WHERE group_id = $1::uuid AND deleted_at IS NULL GROUP BY category`, groupID)
	if err != nil {
		s.logger.Errorw("close: query categories", "error", err)
		return nil, types.NewServerError()
	}
	for crows.Next() {
		var category string
		var sum int64
		if err := crows.Scan(&category, &sum); err != nil {
			crows.Close()
			s.logger.Errorw("close: scan category", "error", err)
			return nil, types.NewServerError()
		}
		spendPerCategory[category] = sum
	}
	crows.Close()
	if err := crows.Err(); err != nil {
		s.logger.Errorw("close: category rows", "error", err)
		return nil, types.NewServerError()
	}

	var total int64
	for _, v := range paid {
		total += v
	}

	shares := fairShares(total, len(members))
	balances := make([]types.MemberBalance, len(members))
	for i, m := range members {
		p := paid[m.membershipID]
		balances[i] = types.MemberBalance{UserID: m.userID, Paid: p, FairShare: shares[i], Net: p - shares[i]}
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

	if _, err := tx.Exec(ctx, `UPDATE groups SET status = 'closed', updated_at = now() WHERE id = $1::uuid`, groupID); err != nil {
		s.logger.Errorw("close: update status", "error", err)
		return nil, types.NewServerError()
	}

	var runID string
	if err := tx.QueryRow(ctx, `INSERT INTO settlement_runs (group_id, computed_by, snapshot) VALUES ($1::uuid, $2::uuid, $3::jsonb) RETURNING id`, groupID, caller.UserID, string(snapJSON)).Scan(&runID); err != nil {
		s.logger.Errorw("close: insert settlement run", "error", err)
		return nil, types.NewServerError()
	}

	for _, t := range plan {
		if _, err := tx.Exec(ctx, `INSERT INTO payments (settlement_run_id, group_id, from_user_id, to_user_id, amount_baisa) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5)`, runID, groupID, t.From, t.To, t.Amount); err != nil {
			s.logger.Errorw("close: insert payment", "error", err)
			return nil, types.NewServerError()
		}
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("close: commit", "error", err)
		return nil, types.NewServerError()
	}

	s.logger.Infow("group closed", "group_id", groupID, "total", total, "transfers", len(plan), "by", caller.UserID)
	return &types.CloseResult{GroupID: groupID, Snapshot: snapshot, Plan: plan}, nil
}
