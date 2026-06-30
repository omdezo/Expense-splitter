package services

import (
	"fmt"
	"testing"

	"expense-splitter/types"
)

func TestFairShares(t *testing.T) {
	cases := []struct {
		total int64
		n     int
		want  []int64
	}{
		{180, 3, []int64{60, 60, 60}},
		{181, 3, []int64{61, 60, 60}},
		{100, 3, []int64{34, 33, 33}},
		{0, 3, []int64{0, 0, 0}},
		{5, 1, []int64{5}},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("total=%d_n=%d", c.total, c.n), func(t *testing.T) {
			got := fairShares(c.total, c.n)
			var sum int64
			for _, s := range got {
				sum += s
			}
			t.Logf("total=%d  n=%d  ->  shares=%v  (sum=%d)", c.total, c.n, got, sum)
			if len(got) != len(c.want) {
				t.Fatalf("len %d, want %d", len(got), len(c.want))
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("idx %d: got %d want %d", i, got[i], c.want[i])
				}
			}
			if sum != c.total {
				t.Errorf("shares sum to %d, want %d", sum, c.total)
			}
		})
	}
}

func TestComputePlanReferenceCase(t *testing.T) {
	balances := []types.MemberBalance{
		{UserID: "ahmed", Net: 40},
		{UserID: "omar", Net: 20},
		{UserID: "mohammed", Net: -60},
	}
	plan := computePlan(balances)

	t.Logf("balances: ahmed +40, omar +20, mohammed -60")
	for _, tr := range plan {
		t.Logf("  %s -> %s : %d", tr.From, tr.To, tr.Amount)
	}

	want := []types.Transfer{
		{From: "mohammed", To: "ahmed", Amount: 40},
		{From: "mohammed", To: "omar", Amount: 20},
	}
	if len(plan) != len(want) {
		t.Fatalf("got %d transfers, want %d", len(plan), len(want))
	}
	for i := range want {
		if plan[i] != want[i] {
			t.Errorf("transfer %d: got %+v want %+v", i, plan[i], want[i])
		}
	}
}

func TestComputePlanReconciles(t *testing.T) {
	cases := [][]types.MemberBalance{
		{{UserID: "a", Net: 30}, {UserID: "b", Net: 10}, {UserID: "c", Net: -25}, {UserID: "d", Net: -15}},
		{{UserID: "a", Net: 100}, {UserID: "b", Net: -100}},
		{{UserID: "a", Net: 0}, {UserID: "b", Net: 0}},
		{{UserID: "a", Net: 50}, {UserID: "b", Net: -20}, {UserID: "c", Net: -30}},
	}
	for ci, bal := range cases {
		t.Run(fmt.Sprintf("case_%d", ci), func(t *testing.T) {
			plan := computePlan(bal)
			net := map[string]int64{}
			line := ""
			for _, b := range bal {
				net[b.UserID] = b.Net
				line += fmt.Sprintf("%s=%+d ", b.UserID, b.Net)
			}
			t.Logf("balances: %s", line)
			if len(plan) == 0 {
				t.Logf("  (no transfers needed)")
			}
			for _, tr := range plan {
				t.Logf("  %s -> %s : %d", tr.From, tr.To, tr.Amount)
				net[tr.From] += tr.Amount
				net[tr.To] -= tr.Amount
			}
			for u, n := range net {
				if n != 0 {
					t.Errorf("%s left with residual %d", u, n)
				}
			}
		})
	}
}
