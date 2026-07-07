package services

import (
	"testing"

	"expense-splitter/types"
)

func TestWeightedShares(t *testing.T) {
	cases := []struct {
		name    string
		total   int64
		weights []int64
		want    []int64
	}{
		{"equal weights = fairShares", 1000, []int64{1, 1, 1}, []int64{334, 333, 333}},
		{"2:1:1 exact", 10000, []int64{2, 1, 1}, []int64{5000, 2500, 2500}},
		{"remainder to earliest", 100, []int64{1, 1, 1}, []int64{34, 33, 33}},
		{"weighted remainder", 101, []int64{2, 1}, []int64{68, 33}},
		{"single participant", 777, []int64{5}, []int64{777}},
		{"zero total", 0, []int64{1, 2}, []int64{0, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := weightedShares(c.total, c.weights)
			var sum int64
			for i, g := range got {
				sum += g
				if g != c.want[i] {
					t.Errorf("share[%d] = %d, want %d (all: %v)", i, g, c.want[i], got)
				}
			}
			if sum != c.total {
				t.Errorf("shares sum to %d, want %d", sum, c.total)
			}
			again := weightedShares(c.total, c.weights)
			for i := range got {
				if got[i] != again[i] {
					t.Errorf("not deterministic at %d: %v vs %v", i, got, again)
				}
			}
			t.Logf("%d split %v -> %v", c.total, c.weights, got)
		})
	}
}

// Bonus #4's core claim: mixed equal + subset + weighted expenses still
// reconcile to zero through the full pipeline (allocations -> fair shares ->
// nets -> transfer plan).
func TestSubsetWeightedReconciles(t *testing.T) {
	members := []string{"a", "b", "c", "d"}
	allocs := []expenseAllocation{
		{Amount: 9000, Users: []string{"a", "b", "c"}, Weights: []int64{1, 1, 1}},  // subset: fuel for the car trio
		{Amount: 10000, Users: []string{"a", "b", "c"}, Weights: []int64{2, 1, 1}}, // weighted: a has the single room
		{Amount: 3000}, // equal across everyone
	}
	paid := map[string]int64{"a": 9000, "b": 10000, "c": 0, "d": 3000}

	fair := buildFairShares(members, allocs)

	var fairSum, total int64
	for _, a := range allocs {
		total += a.Amount
	}
	for _, m := range members {
		fairSum += fair[m]
	}
	if fairSum != total {
		t.Fatalf("fair shares sum to %d, want the total %d", fairSum, total)
	}

	want := map[string]int64{"a": 3000 + 5000 + 750, "b": 3000 + 2500 + 750, "c": 3000 + 2500 + 750, "d": 750}
	for m, w := range want {
		if fair[m] != w {
			t.Errorf("fair[%s] = %d, want %d", m, fair[m], w)
		}
	}

	balances := make([]types.MemberBalance, len(members))
	for i, m := range members {
		balances[i] = types.MemberBalance{UserID: m, Paid: paid[m], FairShare: fair[m], Net: paid[m] - fair[m]}
	}
	plan := computePlan(balances)

	net := map[string]int64{}
	for _, b := range balances {
		net[b.UserID] = b.Net
	}
	var out, in int64
	for _, tr := range plan {
		net[tr.From] += tr.Amount
		net[tr.To] -= tr.Amount
		out += tr.Amount
		in += tr.Amount
		t.Logf("  %s -> %s : %d", tr.From, tr.To, tr.Amount)
	}
	if out != in {
		t.Errorf("sum out %d != sum in %d", out, in)
	}
	for m, n := range net {
		if n != 0 {
			t.Errorf("member %s does not reconcile to zero after the plan: %d", m, n)
		}
	}
	t.Logf("fair=%v — reconciles to zero across equal+subset+weighted", fair)
}

// A member excluded from every subset owes nothing from those expenses.
func TestSubsetExcludesNonParticipants(t *testing.T) {
	members := []string{"a", "b", "c"}
	allocs := []expenseAllocation{
		{Amount: 6000, Users: []string{"a", "b"}, Weights: []int64{1, 1}},
	}
	fair := buildFairShares(members, allocs)
	if fair["c"] != 0 {
		t.Errorf("non-participant c owes %d, want 0", fair["c"])
	}
	if fair["a"] != 3000 || fair["b"] != 3000 {
		t.Errorf("participants owe %d/%d, want 3000/3000", fair["a"], fair["b"])
	}
	t.Logf("subset of {a,b}: fair=%v — c untouched", fair)
}
