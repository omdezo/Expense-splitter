package services

import (
	"testing"

	"expense-splitter/types"
)

var (
	debtor        = paymentActor{IsDebtor: true}
	debtorAdmin   = paymentActor{IsDebtor: true, IsGroupAdmin: true}
	creditor      = paymentActor{IsCreditor: true}
	creditorAdmin = paymentActor{IsCreditor: true, IsGroupAdmin: true}
	groupAdmin    = paymentActor{IsGroupAdmin: true}
	globalAdmin   = paymentActor{IsGlobalAdmin: true}
	bystander     = paymentActor{}
)

var allStatuses = []types.PaymentStatus{
	types.PaymentPending,
	types.PaymentProofSubmitted,
	types.PaymentCreditorConfirmed,
	types.PaymentDisputed,
	types.PaymentSettled,
}

func TestDebtorCanNeverReachSettled(t *testing.T) {
	actions := []paymentAction{actionSubmitProof, actionConfirm, actionDispute, actionFinalize, actionReject}
	for _, actor := range []paymentActor{debtor, debtorAdmin} {
		for _, st := range allStatuses {
			for _, a := range actions {
				next, apiErr := validatePaymentTransition(a, actor, st)
				if apiErr == nil && next == types.PaymentSettled {
					t.Errorf("TAMPER: debtor (groupAdmin=%v) reached settled via %s from %s", actor.IsGroupAdmin, a, st)
				}
			}
		}
	}
	t.Log("no (action, status) pair lets a debtor — even a group-admin debtor — reach settled")
}

func TestDebtorAdvancesOnlyToProofSubmitted(t *testing.T) {
	cases := []struct {
		status types.PaymentStatus
		ok     bool
	}{
		{types.PaymentPending, true},
		{types.PaymentDisputed, true},
		{types.PaymentProofSubmitted, false},
		{types.PaymentCreditorConfirmed, false},
		{types.PaymentSettled, false},
	}
	for _, c := range cases {
		next, apiErr := validatePaymentTransition(actionSubmitProof, debtor, c.status)
		if c.ok {
			if apiErr != nil {
				t.Errorf("submit_proof from %s should succeed, got %v", c.status, apiErr.Message)
			} else if next != types.PaymentProofSubmitted {
				t.Errorf("submit_proof from %s should yield proof_submitted, got %s", c.status, next)
			}
			t.Logf("debtor: %s -> proof_submitted OK", c.status)
		} else if apiErr == nil {
			t.Errorf("submit_proof from %s must be rejected", c.status)
		}
	}
}

func TestCreditorTransitions(t *testing.T) {
	if next, apiErr := validatePaymentTransition(actionConfirm, creditor, types.PaymentProofSubmitted); apiErr != nil || next != types.PaymentCreditorConfirmed {
		t.Errorf("creditor confirm from proof_submitted should give creditor_confirmed, got %v %v", next, apiErr)
	}
	if next, apiErr := validatePaymentTransition(actionDispute, creditor, types.PaymentProofSubmitted); apiErr != nil || next != types.PaymentDisputed {
		t.Errorf("creditor dispute from proof_submitted should give disputed, got %v %v", next, apiErr)
	}
	for _, st := range []types.PaymentStatus{types.PaymentPending, types.PaymentCreditorConfirmed, types.PaymentDisputed, types.PaymentSettled} {
		if _, apiErr := validatePaymentTransition(actionConfirm, creditor, st); apiErr == nil {
			t.Errorf("creditor confirm from %s must be rejected", st)
		}
	}
	for _, st := range allStatuses {
		if _, apiErr := validatePaymentTransition(actionFinalize, creditor, st); apiErr == nil {
			t.Errorf("creditor (non-admin) finalize from %s must be rejected", st)
		}
	}
	t.Log("creditor: confirm/dispute only from proof_submitted; can never finalize")
}

func TestGroupAdminFinalize(t *testing.T) {
	if next, apiErr := validatePaymentTransition(actionFinalize, groupAdmin, types.PaymentCreditorConfirmed); apiErr != nil || next != types.PaymentSettled {
		t.Errorf("group-admin finalize from creditor_confirmed should settle, got %v %v", next, apiErr)
	}
	t.Log("group-admin: creditor_confirmed -> settled OK")
	for _, st := range []types.PaymentStatus{types.PaymentPending, types.PaymentProofSubmitted, types.PaymentDisputed, types.PaymentSettled} {
		if _, apiErr := validatePaymentTransition(actionFinalize, groupAdmin, st); apiErr == nil {
			t.Errorf("group-admin finalize from %s must be rejected (two keys required)", st)
		}
	}
	t.Log("group-admin cannot settle without the creditor's key (incl. disputed payments)")

	if next, apiErr := validatePaymentTransition(actionFinalize, creditorAdmin, types.PaymentCreditorConfirmed); apiErr != nil || next != types.PaymentSettled {
		t.Errorf("creditor who is also group-admin may finalize after confirming, got %v %v", next, apiErr)
	}
}

func TestGlobalAdminOverride(t *testing.T) {
	for _, st := range []types.PaymentStatus{types.PaymentPending, types.PaymentProofSubmitted, types.PaymentCreditorConfirmed, types.PaymentDisputed} {
		if next, apiErr := validatePaymentTransition(actionFinalize, globalAdmin, st); apiErr != nil || next != types.PaymentSettled {
			t.Errorf("global admin override-confirm from %s should settle, got %v %v", st, next, apiErr)
		}
		if next, apiErr := validatePaymentTransition(actionReject, globalAdmin, st); apiErr != nil || next != types.PaymentDisputed {
			t.Errorf("global admin override-reject from %s should dispute, got %v %v", st, next, apiErr)
		}
	}
	t.Log("global admin: override confirm/reject works at every non-settled step")
}

func TestSettledIsTerminal(t *testing.T) {
	actions := []paymentAction{actionSubmitProof, actionConfirm, actionDispute, actionFinalize, actionReject}
	actors := []paymentActor{debtor, creditor, groupAdmin, globalAdmin}
	for _, actor := range actors {
		for _, a := range actions {
			if _, apiErr := validatePaymentTransition(a, actor, types.PaymentSettled); apiErr == nil {
				t.Errorf("settled must be terminal: %s by %+v succeeded", a, actor)
			}
		}
	}
	t.Log("settled is terminal for every actor including the global admin")
}

func TestBystanderDeniedEverything(t *testing.T) {
	actions := []paymentAction{actionSubmitProof, actionConfirm, actionDispute, actionFinalize, actionReject}
	for _, a := range actions {
		for _, st := range allStatuses {
			if _, apiErr := validatePaymentTransition(a, bystander, st); apiErr == nil {
				t.Errorf("plain member must be denied %s from %s", a, st)
			}
		}
	}
	t.Log("a member who is neither party nor admin can do nothing")
}
