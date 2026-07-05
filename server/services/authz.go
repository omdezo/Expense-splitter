package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

type Authorizer struct {
	q      *repo.Queries
	logger *types.Logger
}

func NewAuthorizer(q *repo.Queries, logger *types.Logger) *Authorizer {
	return &Authorizer{q: q, logger: logger}
}

func (a *Authorizer) RequireGlobalAdmin(p *types.Principal) types.APIError {
	if p.IsGlobalAdmin {
		return nil
	}
	return types.NewForbiddenError("global admin only")
}

func (a *Authorizer) RequireVerified(p *types.Principal) types.APIError {
	if p.IsVerified() {
		return nil
	}
	return types.NewForbiddenError("account is not verified")
}

func (a *Authorizer) RequireGroupRole(ctx context.Context, p *types.Principal, groupID string, need types.MembershipRole) types.APIError {
	row, err := a.q.GetMembership(ctx, repo.GetMembershipParams{GroupID: groupID, UserID: p.UserID})
	m := types.Membership{Role: row.Role, Status: row.Status}
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
