package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
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
	qtx := s.q.WithTx(tx)

	row, err := qtx.CreateGroup(ctx, repo.CreateGroupParams{
		Name:                req.Name,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		ExpectedMemberCount: req.ExpectedMemberCount,
		CreatedBy:           caller.UserID,
	})
	if err != nil {
		s.logger.Errorw("create group: insert group", "error", err)
		return nil, types.NewServerError()
	}

	if err := qtx.CreateGroupAdminMembership(ctx, repo.CreateGroupAdminMembershipParams{
		GroupID: row.ID,
		UserID:  adminUserID,
	}); err != nil {
		s.logger.Errorw("create group: insert group admin membership", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("create group: commit", "error", err)
		return nil, types.NewServerError()
	}

	return &types.Group{
		ID:                  row.ID,
		Name:                row.Name,
		StartDate:           row.StartDate,
		EndDate:             row.EndDate,
		Status:              row.Status,
		InviteToken:         row.InviteToken,
		ExpectedMemberCount: row.ExpectedMemberCount,
		CreatedBy:           row.CreatedBy,
		CreatedAt:           row.CreatedAt,
	}, nil
}

func (s *Services) UpdateGroup(ctx context.Context, id types.Identity, groupID string, req types.UpdateGroupRequest) (*types.Group, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("update group: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	status, err := s.q.GetGroupStatus(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("update group: load status", "error", err)
		return nil, types.NewServerError()
	}
	if status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}

	row, err := s.q.UpdateOpenGroup(ctx, repo.UpdateOpenGroupParams{
		Name:                req.Name,
		StartDate:           req.StartDate,
		EndDate:             req.EndDate,
		ExpectedMemberCount: req.ExpectedMemberCount,
		ID:                  groupID,
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// The status=open guard failed between the check above and here — the
		// group was closed concurrently.
		return nil, types.NewConflictError("group is not open")
	case err != nil:
		s.logger.Errorw("update group: update", "error", err)
		return nil, types.NewServerError()
	}
	return &types.Group{
		ID:                  row.ID,
		Name:                row.Name,
		StartDate:           row.StartDate,
		EndDate:             row.EndDate,
		Status:              row.Status,
		InviteToken:         row.InviteToken,
		ExpectedMemberCount: row.ExpectedMemberCount,
		CreatedBy:           row.CreatedBy,
		CreatedAt:           row.CreatedAt,
	}, nil
}

func (s *Services) ListMyGroups(ctx context.Context, id types.Identity) ([]types.GroupListItem, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("list my groups: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	rows, err := s.q.ListGroupsForUser(ctx, caller.UserID)
	if err != nil {
		s.logger.Errorw("list my groups: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.GroupListItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.GroupListItem{
			Group: types.Group{
				ID:                  r.ID,
				Name:                r.Name,
				StartDate:           r.StartDate,
				EndDate:             r.EndDate,
				Status:              r.Status,
				InviteToken:         r.InviteToken,
				ExpectedMemberCount: r.ExpectedMemberCount,
				CreatedBy:           r.CreatedBy,
				CreatedAt:           r.CreatedAt,
			},
			Role:             r.Role,
			MembershipStatus: r.MembershipStatus,
		})
	}
	return out, nil
}

func (s *Services) GetGroup(ctx context.Context, id types.Identity, groupID string) (*types.GroupDetail, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("get group: resolve caller", "error", err)
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
		s.logger.Errorw("get group: load group", "error", err)
		return nil, types.NewServerError()
	}

	members, err := s.q.ListGroupMembers(ctx, groupID)
	if err != nil {
		s.logger.Errorw("get group: query members", "error", err)
		return nil, types.NewServerError()
	}

	detail := &types.GroupDetail{
		Group: types.Group{
			ID:                  g.ID,
			Name:                g.Name,
			StartDate:           g.StartDate,
			EndDate:             g.EndDate,
			Status:              g.Status,
			InviteToken:         g.InviteToken,
			StatusToken:         g.StatusToken,
			ExpectedMemberCount: g.ExpectedMemberCount,
			CreatedBy:           g.CreatedBy,
			CreatedAt:           g.CreatedAt,
		},
		Members: make([]types.MembershipView, 0, len(members)),
	}
	for _, m := range members {
		detail.Members = append(detail.Members, types.MembershipView{
			GroupID:   groupID,
			UserID:    m.UserID,
			Email:     m.Email,
			Role:      m.Role,
			Status:    m.Status,
			CreatedAt: m.CreatedAt,
		})
	}
	return detail, nil
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
