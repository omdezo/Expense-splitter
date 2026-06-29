package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

type Authorizer interface {
	RequireGroupRole(ctx context.Context, p *types.Principal, groupID string, need types.MembershipRole) types.APIError
	RequireGlobalAdmin(p *types.Principal) types.APIError
}

type authorizer struct {
	db     *types.DBPool
	logger *types.Logger
}

func NewAuthorizer(db *types.DBPool, logger *types.Logger) Authorizer {
	return &authorizer{db: db, logger: logger}
}

func (a *authorizer) RequireGlobalAdmin(p *types.Principal) types.APIError {
	if p.IsGlobalAdmin {
		return nil
	}
	return types.NewForbiddenError("global admin only")
}

func (a *authorizer) RequireGroupRole(ctx context.Context, p *types.Principal, groupID string, need types.MembershipRole) types.APIError {
	m, err := a.membership(ctx, groupID, p.UserID)
	found := true
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		found = false
	case err != nil:
		a.logger.Errorw("authz: load membership", "error", err)
		return types.NewServerError()
	}
	return decideGroupRole(p, m, found, need)
}

func decideGroupRole(p *types.Principal, m types.Membership, found bool, need types.MembershipRole) types.APIError {
	if p.IsGlobalAdmin {
		return nil
	}
	if !found {
		return types.NewForbiddenError("not a member of this group")
	}
	if !m.Active() {
		return types.NewForbiddenError("membership not approved")
	}
	if !m.Satisfies(need) {
		return types.NewForbiddenError("insufficient group role")
	}
	return nil
}

func (a *authorizer) membership(ctx context.Context, groupID, userID string) (types.Membership, error) {
	var m types.Membership
	err := a.db.QueryRow(ctx,
		`SELECT role, status FROM memberships WHERE group_id = $1::uuid AND user_id = $2::uuid`,
		groupID, userID).Scan(&m.Role, &m.Status)
	return m, err
}
