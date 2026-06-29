package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

type GroupAuthorizer interface {
	RequireGroupRole(ctx context.Context, p *types.Principal, groupID string, need types.MembershipRole) types.APIError
}

type Authorizer struct {
	db     *types.DBPool
	logger *types.Logger
}

func NewAuthorizer(db *types.DBPool, logger *types.Logger) *Authorizer {
	return &Authorizer{db: db, logger: logger}
}

func (a *Authorizer) RequireGroupRole(ctx context.Context, p *types.Principal, groupID string, need types.MembershipRole) types.APIError {
	if p.IsGlobalAdmin {
		return nil
	}

	m, err := a.membership(ctx, groupID, p.UserID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewForbiddenError("not a member of this group")
	case err != nil:
		a.logger.Errorw("authz: load membership", "error", err)
		return types.NewServerError()
	}

	if !m.Active() {
		return types.NewForbiddenError("membership not approved")
	}
	if !m.Satisfies(need) {
		return types.NewForbiddenError("insufficient group role")
	}
	return nil
}

func (a *Authorizer) membership(ctx context.Context, groupID, userID string) (types.Membership, error) {
	var m types.Membership
	err := a.db.QueryRow(ctx,
		`SELECT role, status FROM memberships WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		groupID, userID).Scan(&m.Role, &m.Status)
	return m, err
}
