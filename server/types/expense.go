package types

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ExpenseCategory string

const (
	CategoryLodging   ExpenseCategory = "lodging"
	CategoryFuel      ExpenseCategory = "fuel"
	CategoryFood      ExpenseCategory = "food"
	CategoryTransport ExpenseCategory = "transport"
	CategoryOther     ExpenseCategory = "other"
)

func (c ExpenseCategory) Valid() bool {
	switch c {
	case CategoryLodging, CategoryFuel, CategoryFood, CategoryTransport, CategoryOther:
		return true
	}
	return false
}

type SplitType string

const (
	SplitEqual    SplitType = "equal"
	SplitSubset   SplitType = "subset"
	SplitWeighted SplitType = "weighted"
)

// ShareInput names a participant of a non-equal split. Weight scales their
// portion (a weight-2 participant owes twice a weight-1 one).
type ShareInput struct {
	UserID string `json:"user_id"`
	Weight int    `json:"weight"`
}

type RecordExpenseRequest struct {
	AmountBaisa int64           `json:"amount_baisa"`
	Category    ExpenseCategory `json:"category"`
	Description string          `json:"description"`
	OccurredOn  string          `json:"occurred_on"`
	// Shares is optional: absent/empty means an equal split across all
	// approved members at settlement. Weight defaults to 1 when omitted.
	Shares []ShareInput `json:"shares,omitempty"`
}

// SplitType derives the split kind from the request's shares.
func (r *RecordExpenseRequest) SplitType() SplitType {
	if len(r.Shares) == 0 {
		return SplitEqual
	}
	for _, s := range r.Shares {
		if s.Weight > 1 {
			return SplitWeighted
		}
	}
	return SplitSubset
}

type UpdateExpenseRequest = RecordExpenseRequest

func (r *RecordExpenseRequest) Validate() APIError {
	if r.AmountBaisa <= 0 {
		return NewBadRequestError("amount_baisa must be a positive integer")
	}
	if !r.Category.Valid() {
		return NewBadRequestError("category must be one of: lodging, fuel, food, transport, other")
	}
	if strings.TrimSpace(r.Description) == "" {
		return NewBadRequestError("description is required")
	}
	if _, err := time.Parse("2006-01-02", r.OccurredOn); err != nil {
		return NewBadRequestError("occurred_on must be a date in YYYY-MM-DD format")
	}
	seen := map[string]bool{}
	for i := range r.Shares {
		s := &r.Shares[i]
		if _, err := uuid.Parse(s.UserID); err != nil {
			return NewBadRequestError("shares[].user_id must be a valid uuid")
		}
		if seen[s.UserID] {
			return NewBadRequestError("shares[].user_id must be unique")
		}
		seen[s.UserID] = true
		if s.Weight == 0 {
			s.Weight = 1
		}
		if s.Weight < 1 || s.Weight > 1000 {
			return NewBadRequestError("shares[].weight must be between 1 and 1000")
		}
	}
	return nil
}

type ExpenseFilter struct {
	Category ExpenseCategory
	PaidBy   string
	Search   string
}

type Expense struct {
	ID          string          `json:"id"`
	GroupID     string          `json:"group_id"`
	PaidBy      string          `json:"paid_by"`
	AmountBaisa int64           `json:"amount_baisa"`
	Category    ExpenseCategory `json:"category"`
	Description string          `json:"description"`
	OccurredOn  string          `json:"occurred_on"`
	SplitType   SplitType       `json:"split_type,omitempty"`
	Shares      []ShareInput    `json:"shares,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}
