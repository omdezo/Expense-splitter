package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
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

	g, err := s.q.GetGroupByInviteToken(ctx, inviteToken)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("invalid invite token")
	case err != nil:
		s.logger.Errorw("join: resolve group", "error", err)
		return nil, types.NewServerError()
	}
	if g.Status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}

	row, err := s.q.CreateJoinRequest(ctx, repo.CreateJoinRequestParams{GroupID: g.ID, UserID: caller.UserID})
	switch {
	case err == nil:
		return &types.MembershipView{
			GroupID:   g.ID,
			UserID:    caller.UserID,
			Email:     caller.Email,
			Role:      row.Role,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
		}, nil
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

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("decide member: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	row, err := qtx.DecideJoinRequest(ctx, repo.DecideJoinRequestParams{
		Status:  decision,
		GroupID: groupID,
		UserID:  targetUserID,
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("no pending join request for this user")
	case err != nil:
		s.logger.Errorw("decide member: update", "error", err)
		return nil, types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     groupID,
		ActorUserID: caller.UserID,
		Action:      "membership." + string(decision),
		Before:      membershipAudit{UserID: targetUserID, Status: types.MembershipRequested},
		After:       membershipAudit{UserID: targetUserID, Status: decision},
	}); err != nil {
		s.logger.Errorw("decide member: write audit", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("decide member: commit", "error", err)
		return nil, types.NewServerError()
	}

	return &types.MembershipView{
		GroupID:   groupID,
		UserID:    targetUserID,
		Role:      row.Role,
		Status:    row.Status,
		CreatedAt: row.CreatedAt,
	}, nil
}

type membershipAudit struct {
	UserID string                 `json:"user_id"`
	Status types.MembershipStatus `json:"status"`
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

	rows, err := s.q.ListJoinRequests(ctx, groupID)
	if err != nil {
		s.logger.Errorw("list requests: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.MembershipView, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.MembershipView{
			GroupID:   groupID,
			UserID:    r.UserID,
			Email:     r.Email,
			Role:      r.Role,
			Status:    r.Status,
			CreatedAt: r.CreatedAt,
		})
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

	target, err := s.q.GetMembership(ctx, repo.GetMembershipParams{GroupID: groupID, UserID: newAdminID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("user is not a member of this group")
	case err != nil:
		s.logger.Errorw("transfer admin: load target", "error", err)
		return nil, types.NewServerError()
	}
	if target.Status != types.MembershipApproved {
		return nil, types.NewBadRequestError("new admin must be an approved member")
	}
	if target.Role == types.RoleGroupAdmin {
		return nil, types.NewConflictError("user is already the group admin")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("transfer admin: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	// Demote the current admin FIRST: the one_group_admin_per_group index forbids
	// two admins, so promoting before demoting would violate it.
	oldAdminID, err := qtx.DemoteGroupAdmin(ctx, groupID)
	if err != nil {
		s.logger.Errorw("transfer admin: demote", "error", err)
		return nil, types.NewServerError()
	}

	promoted, err := qtx.PromoteToGroupAdmin(ctx, repo.PromoteToGroupAdminParams{GroupID: groupID, UserID: newAdminID})
	if err != nil {
		s.logger.Errorw("transfer admin: promote", "error", err)
		return nil, types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     groupID,
		ActorUserID: caller.UserID,
		Action:      "group.admin_transferred",
		Before:      map[string]string{"group_admin": oldAdminID},
		After:       map[string]string{"group_admin": newAdminID},
	}); err != nil {
		s.logger.Errorw("transfer admin: write audit", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("transfer admin: commit", "error", err)
		return nil, types.NewServerError()
	}

	s.logger.Infow("group admin transferred", "group_id", groupID, "new_admin", newAdminID, "by", caller.UserID)
	return &types.MembershipView{
		GroupID:   groupID,
		UserID:    newAdminID,
		Role:      promoted.Role,
		Status:    promoted.Status,
		CreatedAt: promoted.CreatedAt,
	}, nil
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

	m, err := s.q.GetMembership(ctx, repo.GetMembershipParams{GroupID: groupID, UserID: targetUserID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("user is not a member of this group")
	case err != nil:
		s.logger.Errorw("remove member: load membership", "error", err)
		return types.NewServerError()
	}

	if m.Role == types.RoleGroupAdmin {
		return types.NewConflictError("the group admin must transfer the role before being removed")
	}

	hasExpenses, err := s.q.MembershipHasExpenses(ctx, repo.MembershipHasExpensesParams{GroupID: groupID, PaidBy: m.ID})
	if err != nil {
		s.logger.Errorw("remove member: check expenses", "error", err)
		return types.NewServerError()
	}
	if hasExpenses {
		return types.NewConflictError("cannot remove a member who has recorded expenses")
	}

	if err := s.q.DeleteMembership(ctx, m.ID); err != nil {
		s.logger.Errorw("remove member: delete", "error", err)
		return types.NewServerError()
	}

	s.logger.Infow("member removed", "group_id", groupID, "user_id", targetUserID, "by", caller.UserID)
	return nil
}
