package services

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

const ActionExpenseAmountChanged = "expense.amount_changed"

type auditEntry struct {
	GroupID     string
	ActorUserID string
	Action      string
	Before      any
	After       any
}

func (s *Services) writeAudit(ctx context.Context, tx pgx.Tx, e auditEntry) error {
	before, err := jsonbArg(e.Before)
	if err != nil {
		return err
	}
	after, err := jsonbArg(e.After)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`INSERT INTO audit_log (group_id, actor_user_id, action, before, after)
		 VALUES ($1::uuid, $2::uuid, $3, $4::jsonb, $5::jsonb)`,
		e.GroupID, e.ActorUserID, e.Action, before, after)
	return err
}

func jsonbArg(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}
