package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) RequestToJoin(ctx context.Context, id types.Identity, inviteToken string) (*types.MembershipView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("join: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireVerified(caller); apiErr != nil {
		return nil, apiErr
	}

	var groupID string
	var status types.GroupStatus
	err = s.db.QueryRow(ctx, `SELECT id, status FROM groups WHERE invite_token = $1::uuid`, inviteToken).Scan(&groupID, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("invalid invite token")
	case err != nil:
		s.logger.Errorw("join: resolve group", "error", err)
		return nil, types.NewServerError()
	}
	if status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}

	v := &types.MembershipView{GroupID: groupID, UserID: caller.UserID, Email: caller.Email}
	const ins = `
INSERT INTO memberships (group_id, user_id)
VALUES ($1::uuid, $2::uuid)
ON CONFLICT (group_id, user_id) DO NOTHING
RETURNING role, status, created_at`
	err = s.db.QueryRow(ctx, ins, groupID, caller.UserID).Scan(&v.Role, &v.Status, &v.CreatedAt)
	switch {
	case err == nil:
		return v, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewConflictError("already a member or a join request is pending")
	default:
		s.logger.Errorw("join: insert membership", "error", err)
		return nil, types.NewServerError()
	}
}

func (s *Services) ApproveMember(ctx context.Context, id types.Identity, groupID, targetUserID string) (*types.MembershipView, types.APIError) {
	return s.setMemberStatus(ctx, id, groupID, targetUserID, types.MembershipApproved)
}

func (s *Services) RejectMember(ctx context.Context, id types.Identity, groupID, targetUserID string) (*types.MembershipView, types.APIError) {
	return s.setMemberStatus(ctx, id, groupID, targetUserID, types.MembershipRejected)
}

func (s *Services) setMemberStatus(ctx context.Context, id types.Identity, groupID, targetUserID string, decision types.MembershipStatus) (*types.MembershipView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("decide member: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	v := &types.MembershipView{GroupID: groupID, UserID: targetUserID}
	const q = `
UPDATE memberships SET status = $1::membership_status, updated_at = now()
WHERE group_id = $2::uuid AND user_id = $3::uuid AND status = 'requested'
RETURNING role, status, created_at`
	err = s.db.QueryRow(ctx, q, decision, groupID, targetUserID).Scan(&v.Role, &v.Status, &v.CreatedAt)
	switch {
	case err == nil:
		return v, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("no pending join request for this user")
	default:
		s.logger.Errorw("decide member: update", "error", err)
		return nil, types.NewServerError()
	}
}

func (s *Services) ListJoinRequests(ctx context.Context, id types.Identity, groupID string) ([]types.MembershipView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("list requests: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	const q = `
SELECT m.user_id, u.email, m.role, m.status, m.created_at
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.group_id = $1::uuid AND m.status = 'requested'
ORDER BY m.created_at`
	rows, err := s.db.Query(ctx, q, groupID)
	if err != nil {
		s.logger.Errorw("list requests: query", "error", err)
		return nil, types.NewServerError()
	}
	defer rows.Close()

	out := []types.MembershipView{}
	for rows.Next() {
		v := types.MembershipView{GroupID: groupID}
		if err := rows.Scan(&v.UserID, &v.Email, &v.Role, &v.Status, &v.CreatedAt); err != nil {
			s.logger.Errorw("list requests: scan", "error", err)
			return nil, types.NewServerError()
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		s.logger.Errorw("list requests: rows", "error", err)
		return nil, types.NewServerError()
	}
	return out, nil
}
