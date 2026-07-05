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
	GroupID  string        `json:"group_id"`
	Payments []PaymentView `json:"payments"`
}
