package types

import (
	"strings"
	"time"
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

type RecordExpenseRequest struct {
	AmountBaisa int64           `json:"amount_baisa"`
	Category    ExpenseCategory `json:"category"`
	Description string          `json:"description"`
	OccurredOn  string          `json:"occurred_on"`
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
	CreatedAt   time.Time       `json:"created_at"`
}
