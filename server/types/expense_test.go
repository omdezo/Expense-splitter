package types

import "testing"

func TestRecordExpenseRequestValidate(t *testing.T) {
	base := func() RecordExpenseRequest {
		return RecordExpenseRequest{AmountBaisa: 1000, Category: CategoryFood, Description: "dinner", OccurredOn: "2025-06-02"}
	}
	cases := []struct {
		name string
		mut  func(*RecordExpenseRequest)
		ok   bool
	}{
		{"valid", func(*RecordExpenseRequest) {}, true},
		{"zero amount", func(r *RecordExpenseRequest) { r.AmountBaisa = 0 }, false},
		{"negative amount", func(r *RecordExpenseRequest) { r.AmountBaisa = -5 }, false},
		{"bad category", func(r *RecordExpenseRequest) { r.Category = "snacks" }, false},
		{"blank description", func(r *RecordExpenseRequest) { r.Description = "  " }, false},
		{"bad date", func(r *RecordExpenseRequest) { r.OccurredOn = "06/02/2025" }, false},
		{"empty date", func(r *RecordExpenseRequest) { r.OccurredOn = "" }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := base()
			c.mut(&r)
			err := r.Validate()
			if c.ok && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.ok && err == nil {
				t.Fatalf("expected invalid, got nil")
			}
		})
	}
}
