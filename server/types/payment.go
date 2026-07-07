package types

import "time"

type PaymentStatus string

const (
	PaymentPending           PaymentStatus = "pending"
	PaymentProofSubmitted    PaymentStatus = "proof_submitted"
	PaymentCreditorConfirmed PaymentStatus = "creditor_confirmed"
	PaymentDisputed          PaymentStatus = "disputed"
	PaymentSettled           PaymentStatus = "settled"
)

type PaymentView struct {
	ID          string        `json:"id"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	AmountBaisa int64         `json:"amount_baisa"`
	Status      PaymentStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
}

type SettlementPlanResponse struct {
	GroupID      string      `json:"group_id"`
	GroupStatus  GroupStatus `json:"group_status"`
	SettledCount int         `json:"settled_count"`
	TotalCount   int         `json:"total_count"`
	// Note explains an empty plan (e.g. every net balance was already zero).
	Note     string        `json:"note,omitempty"`
	Payments []PaymentView `json:"payments"`
	// Snapshot is the state settlement was computed over: total spent, member
	// count, and each member's paid / fair share / net.
	Snapshot *SettlementSnapshot `json:"snapshot,omitempty"`
}
