package types

import "time"

type MemberSummary struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Paid      int64  `json:"paid"`
	FairShare int64  `json:"fair_share"`
	Net       int64  `json:"net"`
}

type GroupSummary struct {
	GroupID          string           `json:"group_id"`
	Name             string           `json:"name"`
	StartDate        time.Time        `json:"start_date"`
	EndDate          time.Time        `json:"end_date"`
	Status           GroupStatus      `json:"status"`
	MemberCount      int              `json:"member_count"`
	TotalSpent       int64            `json:"total_spent"`
	SpendPerCategory map[string]int64 `json:"spend_per_category"`
	Members          []MemberSummary  `json:"members"`
}

// PublicGroupStatus is the ONLY unauthenticated view of a group: no ids, no
// tokens, no per-person financial detail.
type PublicGroupStatus struct {
	Name        string      `json:"name"`
	Status      GroupStatus `json:"status"`
	TotalSpent  int64       `json:"total_spent"`
	MemberCount int         `json:"member_count"`
}
