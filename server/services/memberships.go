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

func (s *Services) TransferAdmin(ctx context.Context, id types.Identity, groupID, newAdminID string) (*types.MembershipView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("transfer admin: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	var role types.MembershipRole
	var status types.MembershipStatus
	err = s.db.QueryRow(ctx, `SELECT role, status FROM memberships WHERE group_id = $1::uuid AND user_id = $2::uuid`, groupID, newAdminID).Scan(&role, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("user is not a member of this group")
	case err != nil:
		s.logger.Errorw("transfer admin: load target", "error", err)
		return nil, types.NewServerError()
	}
	if status != types.MembershipApproved {
		return nil, types.NewBadRequestError("new admin must be an approved member")
	}
	if role == types.RoleGroupAdmin {
		return nil, types.NewConflictError("user is already the group admin")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("transfer admin: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)

	// Demote the current admin FIRST: the one_group_admin_per_group index forbids
	// two admins, so promoting before demoting would violate it.
	if _, err := tx.Exec(ctx, `UPDATE memberships SET role = 'member', updated_at = now() WHERE group_id = $1::uuid AND role = 'group_admin'`, groupID); err != nil {
		s.logger.Errorw("transfer admin: demote", "error", err)
		return nil, types.NewServerError()
	}

	v := &types.MembershipView{GroupID: groupID, UserID: newAdminID}
	if err := tx.QueryRow(ctx, `UPDATE memberships SET role = 'group_admin', updated_at = now() WHERE group_id = $1::uuid AND user_id = $2::uuid AND status = 'approved' RETURNING role, status, created_at`, groupID, newAdminID).
		Scan(&v.Role, &v.Status, &v.CreatedAt); err != nil {
		s.logger.Errorw("transfer admin: promote", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("transfer admin: commit", "error", err)
		return nil, types.NewServerError()
	}

	s.logger.Infow("group admin transferred", "group_id", groupID, "new_admin", newAdminID, "by", caller.UserID)
	return v, nil
}

func (s *Services) RemoveMember(ctx context.Context, id types.Identity, groupID, targetUserID string) types.APIError {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("remove member: resolve caller", "error", err)
		return types.NewServerError()
	}

	// Removing someone else needs group-admin (global admin passes implicitly);
	// removing yourself is allowed for any member.
	if targetUserID != caller.UserID {
		if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
			return apiErr
		}
	}

	var membershipID string
	var role types.MembershipRole
	err = s.db.QueryRow(ctx, `SELECT id, role FROM memberships WHERE group_id = $1::uuid AND user_id = $2::uuid`, groupID, targetUserID).Scan(&membershipID, &role)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("user is not a member of this group")
	case err != nil:
		s.logger.Errorw("remove member: load membership", "error", err)
		return types.NewServerError()
	}

	if role == types.RoleGroupAdmin {
		return types.NewConflictError("the group admin must transfer the role before being removed")
	}

	var hasExpenses bool
	if err := s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM expenses WHERE group_id = $1::uuid AND paid_by = $2::uuid)`, groupID, membershipID).Scan(&hasExpenses); err != nil {
		s.logger.Errorw("remove member: check expenses", "error", err)
		return types.NewServerError()
	}
	if hasExpenses {
		return types.NewConflictError("cannot remove a member who has recorded expenses")
	}

	if _, err := s.db.Exec(ctx, `DELETE FROM memberships WHERE id = $1::uuid`, membershipID); err != nil {
		s.logger.Errorw("remove member: delete", "error", err)
		return types.NewServerError()
	}

	s.logger.Infow("member removed", "group_id", groupID, "user_id", targetUserID, "by", caller.UserID)
	return nil
}
