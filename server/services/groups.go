package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) CreateGroup(ctx context.Context, id types.Identity, req types.CreateGroupRequest) (*types.Group, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("create group: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	adminUserID, apiErr := s.resolveGroupAdmin(ctx, caller, req.AdminUserID)
	if apiErr != nil {
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
	if err := tx.QueryRow(ctx, insGroup, req.Name, req.StartDate, req.EndDate, req.ExpectedMemberCount, caller.UserID).
		Scan(&g.ID, &g.Name, &g.StartDate, &g.EndDate, &g.Status, &g.InviteToken, &g.ExpectedMemberCount, &g.CreatedBy, &g.CreatedAt); err != nil {
		s.logger.Errorw("create group: insert group", "error", err)
		return nil, types.NewServerError()
	}

	const insMembership = `
INSERT INTO memberships (group_id, user_id, role, status)
VALUES ($1::uuid, $2::uuid, 'group_admin', 'approved')`
	if _, err := tx.Exec(ctx, insMembership, g.ID, adminUserID); err != nil {
		s.logger.Errorw("create group: insert group admin membership", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("create group: commit", "error", err)
		return nil, types.NewServerError()
	}

	return g, nil
}

func (s *Services) resolveGroupAdmin(ctx context.Context, caller *types.Principal, assignedID *string) (string, types.APIError) {
	if caller.IsGlobalAdmin {
		if assignedID == nil {
			return "", types.NewBadRequestError("global admin must assign a group admin via admin_user_id")
		}
		if *assignedID == caller.UserID {
			return "", types.NewBadRequestError("global admin cannot assign themselves as group admin")
		}
		target, err := s.principalByUserID(ctx, *assignedID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return "", types.NewNotFoundError("assigned user not found")
		case err != nil:
			s.logger.Errorw("create group: resolve assigned admin", "error", err)
			return "", types.NewServerError()
		}
		if !target.IsVerified() {
			return "", types.NewBadRequestError("assigned group admin must be a verified user")
		}
		return target.UserID, nil
	}

	if assignedID != nil {
		return "", types.NewForbiddenError("only the global admin may assign a group admin to another member")
	}
	if apiErr := s.authz.RequireVerified(caller); apiErr != nil {
		return "", apiErr
	}
	return caller.UserID, nil
}

func (s *Services) principalByUserID(ctx context.Context, userID string) (*types.Principal, error) {
	const q = `SELECT id, email, is_global_admin, verification_status FROM users WHERE id = $1::uuid`
	p := &types.Principal{}
	if err := s.db.QueryRow(ctx, q, userID).Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus); err != nil {
		return nil, err
	}
	return p, nil
}
