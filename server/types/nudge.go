package types

type NudgeType string

const (
	NudgeDebtor   NudgeType = "debtor"
	NudgeCreditor NudgeType = "creditor"
)

type Nudge struct {
	PaymentID       string    `json:"payment_id"`
	RecipientUserID string    `json:"recipient_user_id"`
	Type            NudgeType `json:"type"`
	AmountBaisa     int64     `json:"amount_baisa"`
}

type NudgeRunResult struct {
	GroupID        string  `json:"group_id"`
	ThresholdHours int     `json:"threshold_hours"`
	Sent           []Nudge `json:"sent"`
	Skipped        int     `json:"skipped"`
}
