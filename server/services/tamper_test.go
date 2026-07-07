package services

import (
	"fmt"
	"testing"

	"expense-splitter/types"
)

// Bonus #1 — tamper-resistant confirmation. These tests prove the three spec
// claims against the FULL transition space, not just happy paths. The global
// admin is exercised separately: their override is spec-sanctioned (req #15)
// and is the only permitted exception.

var tamperActions = []paymentAction{actionSubmitProof, actionConfirm, actionDispute, actionFinalize, actionReject}

func actorLabel(a paymentActor) string {
	return fmt.Sprintf("debtor=%v creditor=%v groupAdmin=%v globalAdmin=%v",
		a.IsDebtor, a.IsCreditor, a.IsGroupAdmin, a.IsGlobalAdmin)
}

// allActorCombos enumerates every role combination a caller could hold,
// EXCLUDING global admin (they are never a payment party by construction and
// hold a sanctioned override tested separately).
func allActorCombos() []paymentActor {
	var out []paymentActor
	for _, d := range []bool{false, true} {
		for _, c := range []bool{false, true} {
			for _, g := range []bool{false, true} {
				if d && c {
					continue // from_user <> to_user is a DB CHECK; impossible
				}
				out = append(out, paymentActor{IsDebtor: d, IsCreditor: c, IsGroupAdmin: g})
			}
		}
	}
	return out
}

// Claim 1: no path lets a debtor mark their own payment settled — no matter
// what other roles they hold.
func TestTamperClaim1DebtorCannotSelfSettle(t *testing.T) {
	checked := 0
	for _, actor := range allActorCombos() {
		if !actor.IsDebtor {
			continue
		}
		for _, st := range allStatuses {
			for _, a := range tamperActions {
				next, apiErr := validatePaymentTransition(a, actor, st)
				checked++
				if apiErr == nil && next == types.PaymentSettled {
					t.Errorf("TAMPER PATH: %s via %s from %s reached settled", actorLabel(actor), a, st)
				}
			}
		}
	}
	t.Logf("claim 1 holds: %d (debtor-role x action x status) combinations, none reaches settled", checked)
}

// Claim 2: settled requires BOTH keys — the creditor's attestation and an
// admin's finalization. Proven structurally over the whole non-override space:
// (a) the ONLY transition that yields settled is finalize-by-admin from
// creditor_confirmed, and (b) the ONLY transition that yields
// creditor_confirmed is confirm-by-creditor from proof_submitted.
func TestTamperClaim2TwoKeysRequired(t *testing.T) {
	for _, actor := range allActorCombos() {
		for _, st := range allStatuses {
			for _, a := range tamperActions {
				next, apiErr := validatePaymentTransition(a, actor, st)
				if apiErr != nil {
					continue
				}
				if next == types.PaymentSettled {
					if !(a == actionFinalize && actor.IsGroupAdmin && !actor.IsDebtor && st == types.PaymentCreditorConfirmed) {
						t.Errorf("settled reached outside the admin-key path: %s via %s from %s", actorLabel(actor), a, st)
					}
				}
				if next == types.PaymentCreditorConfirmed {
					if !(a == actionConfirm && actor.IsCreditor && st == types.PaymentProofSubmitted) {
						t.Errorf("creditor_confirmed reached outside the creditor-key path: %s via %s from %s", actorLabel(actor), a, st)
					}
				}
			}
		}
	}
	t.Log("claim 2 holds: settled only via admin finalize from creditor_confirmed; creditor_confirmed only via creditor confirm")
}

// Claim 3: a disputed payment cannot be reported as settled (only the global
// admin's sanctioned override may resolve a dispute directly).
func TestTamperClaim3DisputedNeverSettled(t *testing.T) {
	for _, actor := range allActorCombos() {
		for _, a := range tamperActions {
			next, apiErr := validatePaymentTransition(a, actor, types.PaymentDisputed)
			if apiErr == nil && next == types.PaymentSettled {
				t.Errorf("disputed settled directly by %s via %s", actorLabel(actor), a)
			}
		}
	}
	// The dispute is resolved only by the debtor re-submitting proof and the
	// full two-key path running again.
	next, apiErr := validatePaymentTransition(actionSubmitProof, debtor, types.PaymentDisputed)
	if apiErr != nil || next != types.PaymentProofSubmitted {
		t.Errorf("disputed must be re-submittable by the debtor, got %v %v", next, apiErr)
	}
	t.Log("claim 3 holds: disputed is only resolvable by re-submitting proof and re-running both keys")
}

// The sanctioned exception, pinned down so it can't silently widen: the global
// admin may override-confirm or override-reject at any non-settled step, and
// nothing else changes for them.
func TestTamperGlobalOverrideIsBounded(t *testing.T) {
	for _, st := range allStatuses {
		for _, a := range tamperActions {
			next, apiErr := validatePaymentTransition(a, globalAdmin, st)
			if st == types.PaymentSettled {
				if apiErr == nil {
					t.Errorf("settled must be terminal even for the global admin (%s)", a)
				}
				continue
			}
			switch a {
			case actionFinalize:
				if apiErr != nil || next != types.PaymentSettled {
					t.Errorf("global override-confirm from %s should settle, got %v %v", st, next, apiErr)
				}
			case actionReject:
				if apiErr != nil || next != types.PaymentDisputed {
					t.Errorf("global override-reject from %s should dispute, got %v %v", st, next, apiErr)
				}
			default:
				if apiErr == nil {
					t.Errorf("global admin must NOT hold the debtor/creditor keys: %s from %s succeeded", a, st)
				}
			}
		}
	}
	t.Log("override bounded: global admin can finalize/reject any non-settled payment and nothing more")
}
