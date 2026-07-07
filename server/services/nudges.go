package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

// RunNudges generates reminders for a partially-settled group (bonus #5):
// debtors sitting on a `pending` payment and creditors sitting on a
// `proof_submitted` one, both older than the threshold. Idempotent by
// construction — the notifications UNIQUE key plus a conditional upsert means
// re-running within the threshold re-sends nothing, no matter how often the
// endpoint is polled. Delivery is the structured log (spec allows email/log).
func (s *Services) RunNudges(ctx context.Context, id types.Identity, groupID string, hours int) (*types.NudgeRunResult, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("nudges: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleGroupAdmin); apiErr != nil {
		return nil, apiErr
	}

	stale, err := s.q.ListStalePayments(ctx, repo.ListStalePaymentsParams{GroupID: groupID, Hours: int32(hours)})
	if err != nil {
		s.logger.Errorw("nudges: list stale payments", "error", err)
		return nil, types.NewServerError()
	}

	result := &types.NudgeRunResult{GroupID: groupID, ThresholdHours: hours, Sent: []types.Nudge{}}
	for _, p := range stale {
		nudge := types.Nudge{PaymentID: p.ID, AmountBaisa: p.AmountBaisa}
		switch p.Status {
		case types.PaymentPending:
			nudge.RecipientUserID, nudge.Type = p.FromUserID, types.NudgeDebtor
		case types.PaymentProofSubmitted:
			nudge.RecipientUserID, nudge.Type = p.ToUserID, types.NudgeCreditor
		default:
			continue
		}

		_, err := s.q.UpsertNudge(ctx, repo.UpsertNudgeParams{
			PaymentID:       nudge.PaymentID,
			RecipientUserID: nudge.RecipientUserID,
			NudgeType:       nudge.Type,
			Hours:           int32(hours),
		})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// Conflict without update: this recipient was nudged for this
			// payment within the threshold — don't spam.
			result.Skipped++
			continue
		case err != nil:
			s.logger.Errorw("nudges: upsert", "error", err)
			return nil, types.NewServerError()
		}

		s.logger.Infow("nudge sent",
			"group_id", groupID,
			"payment_id", nudge.PaymentID,
			"recipient", nudge.RecipientUserID,
			"type", nudge.Type,
			"amount_baisa", nudge.AmountBaisa,
		)
		result.Sent = append(result.Sent, nudge)
	}
	return result, nil
}
