package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

func (s *Services) GetSettlementPlan(ctx context.Context, id types.Identity, groupID string) (*types.SettlementPlanResponse, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("settlement plan: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleMember); apiErr != nil {
		return nil, apiErr
	}

	status, err := s.q.GetGroupStatus(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("settlement plan: load status", "error", err)
		return nil, types.NewServerError()
	}
	if status == types.GroupOpen {
		return nil, types.NewConflictError("group is not closed yet — no settlement plan exists")
	}

	rows, err := s.q.ListPayments(ctx, groupID)
	if err != nil {
		s.logger.Errorw("settlement plan: query payments", "error", err)
		return nil, types.NewServerError()
	}

	resp := &types.SettlementPlanResponse{
		GroupID:     groupID,
		GroupStatus: status,
		TotalCount:  len(rows),
		Payments:    make([]types.PaymentView, 0, len(rows)),
	}
	for _, r := range rows {
		if r.Status == types.PaymentSettled {
			resp.SettledCount++
		}
		resp.Payments = append(resp.Payments, types.PaymentView{
			ID:          r.ID,
			From:        r.FromUserID,
			To:          r.ToUserID,
			AmountBaisa: r.AmountBaisa,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
		})
	}

	// Attach the snapshot settlement was computed over, so the response
	// explains itself — especially when the plan is empty.
	raw, err := s.q.GetSettlementSnapshot(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// closed-but-unsettled groups always have a run; tolerate absence.
	case err != nil:
		s.logger.Errorw("settlement plan: load snapshot", "error", err)
		return nil, types.NewServerError()
	default:
		var snap types.SettlementSnapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			s.logger.Errorw("settlement plan: decode snapshot", "error", err)
		} else {
			resp.Snapshot = &snap
		}
	}

	if resp.TotalCount == 0 {
		if resp.Snapshot != nil && resp.Snapshot.MemberCount <= 1 {
			resp.Note = "no transfers were needed: the group had a single member, so their fair share equals what they paid"
		} else {
			resp.Note = "no transfers were needed: every member's net balance was already zero at close"
		}
	}
	return resp, nil
}

type paymentAction string

const (
	actionSubmitProof paymentAction = "submit_proof"
	actionConfirm     paymentAction = "confirm"
	actionDispute     paymentAction = "dispute"
	actionFinalize    paymentAction = "finalize"
	actionReject      paymentAction = "reject"
)

type paymentActor struct {
	IsDebtor      bool
	IsCreditor    bool
	IsGroupAdmin  bool
	IsGlobalAdmin bool
}

// validatePaymentTransition is the single decision point of the confirmation
// state machine (req #15). The tamper-resistance spine (bonus #1) lives in the
// ORDER of the checks: the debtor self-marking ban is decided before any admin
// power is considered, so no role combination lets a debtor advance their own
// payment past proof_submitted.
func validatePaymentTransition(action paymentAction, actor paymentActor, status types.PaymentStatus) (types.PaymentStatus, types.APIError) {
	if status == types.PaymentSettled {
		return "", types.NewConflictError("payment is already settled")
	}

	switch action {
	case actionSubmitProof:
		if !actor.IsDebtor {
			return "", types.NewForbiddenError("only the debtor may submit proof")
		}
		if status != types.PaymentPending && status != types.PaymentDisputed {
			return "", types.NewConflictError("proof can only be submitted for a pending or disputed payment")
		}
		return types.PaymentProofSubmitted, nil

	case actionConfirm:
		if !actor.IsCreditor {
			return "", types.NewForbiddenError("only the creditor may confirm receipt")
		}
		if status != types.PaymentProofSubmitted {
			return "", types.NewConflictError("only a submitted proof can be confirmed")
		}
		return types.PaymentCreditorConfirmed, nil

	case actionDispute:
		if !actor.IsCreditor {
			return "", types.NewForbiddenError("only the creditor may dispute receipt")
		}
		if status != types.PaymentProofSubmitted {
			return "", types.NewConflictError("only a submitted proof can be disputed")
		}
		return types.PaymentDisputed, nil

	case actionFinalize:
		if actor.IsDebtor {
			return "", types.NewForbiddenError("a debtor cannot settle their own payment")
		}
		if !actor.IsGroupAdmin && !actor.IsGlobalAdmin {
			return "", types.NewForbiddenError("only a group admin or the global admin may finalize")
		}
		if actor.IsGlobalAdmin {
			return types.PaymentSettled, nil
		}
		if status != types.PaymentCreditorConfirmed {
			return "", types.NewConflictError("payment requires the creditor's confirmation before it can be finalized")
		}
		return types.PaymentSettled, nil

	case actionReject:
		if !actor.IsGroupAdmin && !actor.IsGlobalAdmin {
			return "", types.NewForbiddenError("only a group admin or the global admin may reject")
		}
		if actor.IsGlobalAdmin {
			return types.PaymentDisputed, nil
		}
		if status != types.PaymentProofSubmitted && status != types.PaymentCreditorConfirmed {
			return "", types.NewConflictError("there is nothing to reject on this payment")
		}
		return types.PaymentDisputed, nil
	}
	return "", types.NewServerError()
}

type paymentStatusAudit struct {
	PaymentID string              `json:"payment_id"`
	Status    types.PaymentStatus `json:"status"`
}

func (s *Services) SubmitProof(ctx context.Context, id types.Identity, paymentID string, req types.SubmitProofRequest) (*types.PaymentView, types.APIError) {
	return s.transitionPayment(ctx, id, paymentID, actionSubmitProof, func(qtx *repo.Queries, p repo.GetPaymentRow) error {
		if err := qtx.UnsetCurrentProof(ctx, paymentID); err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(req.Note))
		hash := hex.EncodeToString(sum[:])
		size := int64(len(req.Note))
		note := req.Note
		_, err := qtx.CreateProof(ctx, repo.CreateProofParams{
			PaymentID: paymentID,
			ProofType: types.ProofText,
			Sha256:    &hash,
			ByteSize:  &size,
			Note:      &note,
		})
		return err
	})
}

func (s *Services) ConfirmPayment(ctx context.Context, id types.Identity, paymentID string) (*types.PaymentView, types.APIError) {
	return s.transitionPayment(ctx, id, paymentID, actionConfirm, nil)
}

func (s *Services) DisputePayment(ctx context.Context, id types.Identity, paymentID string) (*types.PaymentView, types.APIError) {
	return s.transitionPayment(ctx, id, paymentID, actionDispute, nil)
}

func (s *Services) FinalizePayment(ctx context.Context, id types.Identity, paymentID string) (*types.PaymentView, types.APIError) {
	return s.transitionPayment(ctx, id, paymentID, actionFinalize, nil)
}

func (s *Services) RejectPayment(ctx context.Context, id types.Identity, paymentID string) (*types.PaymentView, types.APIError) {
	return s.transitionPayment(ctx, id, paymentID, actionReject, nil)
}

// transitionPayment runs one state-machine step in a single transaction:
// resolve actor roles, validate the transition, apply any side effect (proof
// insert), then the version-guarded UPDATE — the optimistic lock that makes
// every transition exactly-once (bonus #3): a concurrent actor who read the
// same version loses the race and gets a 409.
func (s *Services) transitionPayment(ctx context.Context, id types.Identity, paymentID string, action paymentAction, sideEffect func(*repo.Queries, repo.GetPaymentRow) error) (*types.PaymentView, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("payment transition: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("payment transition: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	p, err := qtx.GetPayment(ctx, paymentID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("payment not found")
	case err != nil:
		s.logger.Errorw("payment transition: load payment", "error", err)
		return nil, types.NewServerError()
	}

	actor := paymentActor{
		IsDebtor:      p.FromUserID == caller.UserID,
		IsCreditor:    p.ToUserID == caller.UserID,
		IsGlobalAdmin: caller.IsGlobalAdmin,
	}
	m, err := qtx.GetMembership(ctx, repo.GetMembershipParams{GroupID: p.GroupID, UserID: caller.UserID})
	switch {
	case err == nil:
		actor.IsGroupAdmin = m.Role == types.RoleGroupAdmin && m.Status == types.MembershipApproved
	case !errors.Is(err, pgx.ErrNoRows):
		s.logger.Errorw("payment transition: load membership", "error", err)
		return nil, types.NewServerError()
	}
	if !actor.IsDebtor && !actor.IsCreditor && !actor.IsGroupAdmin && !actor.IsGlobalAdmin {
		return nil, types.NewForbiddenError("you are not a participant in this payment")
	}

	newStatus, apiErr := validatePaymentTransition(action, actor, p.Status)
	if apiErr != nil {
		return nil, apiErr
	}

	if sideEffect != nil {
		if err := sideEffect(qtx, p); err != nil {
			s.logger.Errorw("payment transition: side effect", "action", string(action), "error", err)
			return nil, types.NewServerError()
		}
	}

	row, err := qtx.TransitionPayment(ctx, repo.TransitionPaymentParams{Status: newStatus, ID: paymentID, Version: p.Version})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewConflictError("payment was modified concurrently — retry")
	case err != nil:
		s.logger.Errorw("payment transition: update", "error", err)
		return nil, types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     p.GroupID,
		ActorUserID: caller.UserID,
		Action:      auditActionFor(action, actor, p.Status),
		Before:      paymentStatusAudit{PaymentID: paymentID, Status: p.Status},
		After:       paymentStatusAudit{PaymentID: paymentID, Status: newStatus},
	}); err != nil {
		s.logger.Errorw("payment transition: write audit", "error", err)
		return nil, types.NewServerError()
	}

	if newStatus == types.PaymentSettled {
		unsettled, err := qtx.CountUnsettledPayments(ctx, p.GroupID)
		if err != nil {
			s.logger.Errorw("payment transition: count unsettled", "error", err)
			return nil, types.NewServerError()
		}
		if unsettled == 0 {
			if err := qtx.MarkGroupSettled(ctx, p.GroupID); err != nil {
				s.logger.Errorw("payment transition: mark group settled", "error", err)
				return nil, types.NewServerError()
			}
			if err := s.writeAudit(ctx, qtx, auditEntry{
				GroupID:     p.GroupID,
				ActorUserID: caller.UserID,
				Action:      "group.fully_settled",
				Before:      map[string]string{"status": string(types.GroupClosed)},
				After:       map[string]string{"status": string(types.GroupSettled)},
			}); err != nil {
				s.logger.Errorw("payment transition: audit group settled", "error", err)
				return nil, types.NewServerError()
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("payment transition: commit", "error", err)
		return nil, types.NewServerError()
	}

	return &types.PaymentView{
		ID:          row.ID,
		From:        row.FromUserID,
		To:          row.ToUserID,
		AmountBaisa: row.AmountBaisa,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
	}, nil
}

// auditActionFor names the event; a global-admin acting from a state the
// group-admin could not act from is an override (req #15) and is labelled so.
func auditActionFor(action paymentAction, actor paymentActor, from types.PaymentStatus) string {
	switch action {
	case actionSubmitProof:
		return "payment.proof_submitted"
	case actionConfirm:
		return "payment.creditor_confirmed"
	case actionDispute:
		return "payment.creditor_disputed"
	case actionFinalize:
		if actor.IsGlobalAdmin && from != types.PaymentCreditorConfirmed {
			return "payment.settled.override"
		}
		return "payment.settled"
	case actionReject:
		if actor.IsGlobalAdmin && from != types.PaymentProofSubmitted && from != types.PaymentCreditorConfirmed {
			return "payment.rejected.override"
		}
		return "payment.rejected"
	}
	return "payment.unknown"
}
