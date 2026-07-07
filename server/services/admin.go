package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"expense-splitter/types"
)

// requireGlobalAdmin resolves the caller and enforces the system-wide role.
func (s *Services) requireGlobalAdmin(ctx context.Context, id types.Identity) (*types.Principal, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("admin: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGlobalAdmin(caller); apiErr != nil {
		return nil, apiErr
	}
	return caller, nil
}

// AdminListUsers lists every account, optionally filtered by verification
// status (spec #3: the global admin manages all users).
func (s *Services) AdminListUsers(ctx context.Context, id types.Identity, status string) ([]types.AdminUserView, types.APIError) {
	if _, apiErr := s.requireGlobalAdmin(ctx, id); apiErr != nil {
		return nil, apiErr
	}

	var filter *types.VerificationStatus
	if status != "" {
		v := types.VerificationStatus(status)
		switch v {
		case types.VerificationRegistered, types.VerificationPending, types.VerificationVerified, types.VerificationRejected:
			filter = &v
		default:
			return nil, types.NewBadRequestError("status must be one of: registered, pending_verification, verified, rejected")
		}
	}

	rows, err := s.q.ListUsers(ctx, filter)
	if err != nil {
		s.logger.Errorw("admin list users: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.AdminUserView, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.AdminUserView{
			ID:                 r.ID,
			Email:              r.Email,
			IsGlobalAdmin:      r.IsGlobalAdmin,
			VerificationStatus: r.VerificationStatus,
			Linked:             r.Linked,
			CreatedAt:          r.CreatedAt,
		})
	}
	return out, nil
}

// AdminGetUser returns one account plus every group membership it holds.
func (s *Services) AdminGetUser(ctx context.Context, id types.Identity, userID string) (*types.AdminUserDetail, types.APIError) {
	if _, apiErr := s.requireGlobalAdmin(ctx, id); apiErr != nil {
		return nil, apiErr
	}

	u, err := s.q.GetUserAdminView(ctx, userID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("user not found")
	case err != nil:
		s.logger.Errorw("admin get user: query", "error", err)
		return nil, types.NewServerError()
	}

	memberships, err := s.q.ListUserMemberships(ctx, userID)
	if err != nil {
		s.logger.Errorw("admin get user: memberships", "error", err)
		return nil, types.NewServerError()
	}

	detail := &types.AdminUserDetail{
		AdminUserView: types.AdminUserView{
			ID:                 u.ID,
			Email:              u.Email,
			IsGlobalAdmin:      u.IsGlobalAdmin,
			VerificationStatus: u.VerificationStatus,
			Linked:             u.Linked,
			CreatedAt:          u.CreatedAt,
		},
		Memberships: make([]types.UserMembershipView, 0, len(memberships)),
	}
	for _, m := range memberships {
		detail.Memberships = append(detail.Memberships, types.UserMembershipView{
			GroupID:   m.GroupID,
			GroupName: m.GroupName,
			Role:      m.Role,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
		})
	}
	return detail, nil
}

// AdminDeleteUser removes an account that has no group footprint. Anyone who
// ever joined a group keeps their row — memberships anchor expenses, payments
// and the audit trail, and deleting them would corrupt the books.
func (s *Services) AdminDeleteUser(ctx context.Context, id types.Identity, userID string) types.APIError {
	caller, apiErr := s.requireGlobalAdmin(ctx, id)
	if apiErr != nil {
		return apiErr
	}
	if caller.UserID == userID {
		return types.NewConflictError("the global admin cannot delete their own account")
	}

	target, err := s.q.GetUserAdminView(ctx, userID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("user not found")
	case err != nil:
		s.logger.Errorw("admin delete user: load", "error", err)
		return types.NewServerError()
	}
	if target.IsGlobalAdmin {
		return types.NewConflictError("the global admin account cannot be deleted")
	}

	memberships, err := s.q.CountUserMemberships(ctx, userID)
	if err != nil {
		s.logger.Errorw("admin delete user: count memberships", "error", err)
		return types.NewServerError()
	}
	if memberships > 0 {
		return types.NewConflictError("cannot delete a user who belongs to groups — remove their memberships first")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("admin delete user: begin tx", "error", err)
		return types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.DeleteUser(ctx, userID); err != nil {
		if isFKViolation(err) {
			return types.NewConflictError("user is referenced by financial history and cannot be deleted")
		}
		s.logger.Errorw("admin delete user: delete", "error", err)
		return types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		ActorUserID: caller.UserID,
		Action:      "user.deleted",
		Before:      map[string]string{"user_id": userID, "email": target.Email},
	}); err != nil {
		s.logger.Errorw("admin delete user: audit", "error", err)
		return types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("admin delete user: commit", "error", err)
		return types.NewServerError()
	}
	s.logger.Infow("user deleted", "user_id", userID, "by", caller.UserID)
	return nil
}

// AdminListGroups is the global admin's view of every group in the system.
func (s *Services) AdminListGroups(ctx context.Context, id types.Identity) ([]types.AdminGroupView, types.APIError) {
	if _, apiErr := s.requireGlobalAdmin(ctx, id); apiErr != nil {
		return nil, apiErr
	}

	rows, err := s.q.ListAllGroups(ctx)
	if err != nil {
		s.logger.Errorw("admin list groups: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.AdminGroupView, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.AdminGroupView{
			ID:          r.ID,
			Name:        r.Name,
			StartDate:   r.StartDate,
			EndDate:     r.EndDate,
			Status:      r.Status,
			CreatedBy:   r.CreatedBy,
			MemberCount: r.MemberCount,
			TotalSpent:  r.TotalSpent,
			CreatedAt:   r.CreatedAt,
		})
	}
	return out, nil
}

// AdminDeleteGroup removes a group that never accumulated history: no
// expenses (even deleted ones), no payments, no settlement run, no audit
// rows. Anything with history is immutable bookkeeping and stays.
func (s *Services) AdminDeleteGroup(ctx context.Context, id types.Identity, groupID string) types.APIError {
	caller, apiErr := s.requireGlobalAdmin(ctx, id)
	if apiErr != nil {
		return apiErr
	}

	g, err := s.q.GetGroupByID(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("admin delete group: load", "error", err)
		return types.NewServerError()
	}

	hasHistory, err := s.q.GroupHasHistory(ctx, groupID)
	if err != nil {
		s.logger.Errorw("admin delete group: history check", "error", err)
		return types.NewServerError()
	}
	if hasHistory {
		return types.NewConflictError("cannot delete a group with expenses, payments or audit history")
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("admin delete group: begin tx", "error", err)
		return types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	if err := qtx.DeleteGroupMemberships(ctx, groupID); err != nil {
		s.logger.Errorw("admin delete group: delete memberships", "error", err)
		return types.NewServerError()
	}
	if err := qtx.DeleteGroup(ctx, groupID); err != nil {
		if isFKViolation(err) {
			return types.NewConflictError("group is referenced by history and cannot be deleted")
		}
		s.logger.Errorw("admin delete group: delete", "error", err)
		return types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		ActorUserID: caller.UserID,
		Action:      "group.deleted",
		Before:      map[string]string{"group_id": groupID, "name": g.Name},
	}); err != nil {
		s.logger.Errorw("admin delete group: audit", "error", err)
		return types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("admin delete group: commit", "error", err)
		return types.NewServerError()
	}
	s.logger.Infow("group deleted", "group_id", groupID, "by", caller.UserID)
	return nil
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
