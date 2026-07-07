package services

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

const ActionExpenseAmountChanged = "expense.amount_changed"

type auditEntry struct {
	GroupID     string
	ActorUserID string
	Action      string
	Before      any
	After       any
}

// writeAudit records an audit row through the SAME queries handle as the change
// it documents — pass the WithTx-bound queries so both commit atomically. An
// empty GroupID records a system-level event (group_id NULL).
func (s *Services) writeAudit(ctx context.Context, q *repo.Queries, e auditEntry) error {
	before, err := jsonbArg(e.Before)
	if err != nil {
		return err
	}
	after, err := jsonbArg(e.After)
	if err != nil {
		return err
	}
	var groupID *string
	if e.GroupID != "" {
		groupID = &e.GroupID
	}
	return q.CreateAuditEntry(ctx, repo.CreateAuditEntryParams{
		GroupID:     groupID,
		ActorUserID: e.ActorUserID,
		Action:      e.Action,
		Before:      before,
		After:       after,
	})
}

func jsonbArg(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// ListGroupAudit is the admin read API for a group's audit trail (req #16).
func (s *Services) ListGroupAudit(ctx context.Context, id types.Identity, groupID string) ([]types.AuditEntryView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("audit list: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	rows, err := s.q.ListAuditEntries(ctx, groupID)
	if err != nil {
		s.logger.Errorw("audit list: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.AuditEntryView, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.AuditEntryView{
			ID:          r.ID,
			ActorUserID: r.ActorUserID,
			Action:      r.Action,
			Before:      r.Before,
			After:       r.After,
			CreatedAt:   r.CreatedAt,
		})
	}
	return out, nil
}
