package types

type MemberBalance struct {
	UserID    string `json:"user_id"`
	Paid      int64  `json:"paid"`
	FairShare int64  `json:"fair_share"`
	Net       int64  `json:"net"`
}

type Transfer struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount int64  `json:"amount"`
}

type SettlementSnapshot struct {
	TotalSpent       int64            `json:"total_spent"`
	MemberCount      int              `json:"member_count"`
	SpendPerCategory map[string]int64 `json:"spend_per_category"`
	Members          []MemberBalance  `json:"members"`
}

type CloseResult struct {
	GroupID  string             `json:"group_id"`
	Snapshot SettlementSnapshot `json:"snapshot"`
	Plan     []Transfer         `json:"plan"`
}
