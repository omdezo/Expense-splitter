package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) CreateGroup(ctx context.Context, id types.Identity, req types.CreateGroupRequest) (*types.Group, types.APIError) {
	p, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("create group: resolve principal", "error", err)
		return nil, types.NewServerError()
	}

	if apiErr := s.authz.RequireVerified(p); apiErr != nil {
		return nil, apiErr
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("create group: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)

	g := &types.Group{}
	const insGroup = `
INSERT INTO groups (name, start_date, end_date, expected_member_count, created_by)
VALUES ($1, $2, $3, $4, $5::uuid)
RETURNING id, name, start_date, end_date, status, invite_token, expected_member_count, created_by, created_at`
	if err := tx.QueryRow(ctx, insGroup, req.Name, req.StartDate, req.EndDate, req.ExpectedMemberCount, p.UserID).
		Scan(&g.ID, &g.Name, &g.StartDate, &g.EndDate, &g.Status, &g.InviteToken, &g.ExpectedMemberCount, &g.CreatedBy, &g.CreatedAt); err != nil {
		s.logger.Errorw("create group: insert group", "error", err)
		return nil, types.NewServerError()
	}

	const insMembership = `
INSERT INTO memberships (group_id, user_id, role, status)
VALUES ($1::uuid, $2::uuid, 'group_admin', 'approved')`
	if _, err := tx.Exec(ctx, insMembership, g.ID, p.UserID); err != nil {
		s.logger.Errorw("create group: insert creator membership", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("create group: commit", "error", err)
		return nil, types.NewServerError()
	}

	return g, nil
}
