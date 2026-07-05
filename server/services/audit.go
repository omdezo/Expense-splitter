package services

import (
	"context"
	"encoding/json"

	"expense-splitter/database/repo"
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
// it documents — pass the WithTx-bound queries so both commit atomically.
func (s *Services) writeAudit(ctx context.Context, q *repo.Queries, e auditEntry) error {
	before, err := jsonbArg(e.Before)
	if err != nil {
		return err
	}
	after, err := jsonbArg(e.After)
	if err != nil {
		return err
	}
	return q.CreateAuditEntry(ctx, repo.CreateAuditEntryParams{
		GroupID:     e.GroupID,
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
