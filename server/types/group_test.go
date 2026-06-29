package types

import (
	"testing"
	"time"
)

func TestCreateGroupRequestValidate(t *testing.T) {
	start := time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2025, 6, 7, 18, 0, 0, 0, time.UTC)
	neg := -1
	zero := 0

	cases := []struct {
		name string
		req  CreateGroupRequest
		ok   bool
	}{
		{"valid", CreateGroupRequest{Name: "Trip", StartDate: start, EndDate: end}, true},
		{"valid with count", CreateGroupRequest{Name: "Trip", StartDate: start, EndDate: end, ExpectedMemberCount: &zero}, true},
		{"blank name", CreateGroupRequest{Name: "   ", StartDate: start, EndDate: end}, false},
		{"missing start", CreateGroupRequest{Name: "Trip", EndDate: end}, false},
		{"missing end", CreateGroupRequest{Name: "Trip", StartDate: start}, false},
		{"end before start", CreateGroupRequest{Name: "Trip", StartDate: end, EndDate: start}, false},
		{"negative count", CreateGroupRequest{Name: "Trip", StartDate: start, EndDate: end, ExpectedMemberCount: &neg}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			if c.ok && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !c.ok && err == nil {
				t.Fatalf("expected invalid, got nil")
			}
		})
	}
}
