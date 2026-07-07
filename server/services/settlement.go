package services

import (
	"sort"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

// allocationsFromRows regroups the flat expense/share join into one
// allocation per expense (rows arrive ordered by expense, then user id).
func allocationsFromRows(rows []repo.ListExpenseAllocationsRow) []expenseAllocation {
	var out []expenseAllocation
	idx := map[string]int{}
	for _, r := range rows {
		i, ok := idx[r.ID]
		if !ok {
			i = len(out)
			idx[r.ID] = i
			out = append(out, expenseAllocation{Amount: r.AmountBaisa})
		}
		if r.UserID != nil {
			w := int64(1)
			if r.Weight != nil {
				w = int64(*r.Weight)
			}
			out[i].Users = append(out[i].Users, *r.UserID)
			out[i].Weights = append(out[i].Weights, w)
		}
	}
	return out
}

// fairShares splits total among n members. total/n is the base; the leftover
// (total % n) baisa each go to the first members in the caller's stable order,
// so the shares always sum exactly to total. Same input -> same output.
func fairShares(total int64, n int) []int64 {
	if n <= 0 {
		return nil
	}
	base := total / int64(n)
	rem := total % int64(n)
	shares := make([]int64, n)
	for i := range shares {
		shares[i] = base
		if int64(i) < rem {
			shares[i]++
		}
	}
	return shares
}

// weightedShares splits total by weight: participant i owes total*w_i/W,
// with the integer remainder distributed one baisa each to the first
// participants in the caller's stable order. weightedShares(t, [1,1,...,1])
// equals fairShares(t, n). Shares always sum exactly to total.
func weightedShares(total int64, weights []int64) []int64 {
	if len(weights) == 0 {
		return nil
	}
	var totalWeight int64
	for _, w := range weights {
		totalWeight += w
	}
	if totalWeight <= 0 {
		return make([]int64, len(weights))
	}
	shares := make([]int64, len(weights))
	var allocated int64
	for i, w := range weights {
		shares[i] = total * w / totalWeight
		allocated += shares[i]
	}
	rem := total - allocated
	for i := 0; rem > 0 && i < len(shares); i++ {
		shares[i]++
		rem--
	}
	return shares
}

// expenseAllocation is one expense's split instruction: an empty Users list
// means "equal across all approved members at settlement" (req #7 fair-share
// semantics); otherwise only the named users owe, scaled by Weights (bonus #4).
type expenseAllocation struct {
	Amount  int64
	Users   []string
	Weights []int64
}

// buildFairShares turns per-expense allocations into each member's total fair
// share. memberIDs must be in stable order (sorted by user id) — remainder
// baisa always land on the earliest participants, so same input -> same output.
func buildFairShares(memberIDs []string, allocs []expenseAllocation) map[string]int64 {
	fair := make(map[string]int64, len(memberIDs))
	for _, id := range memberIDs {
		fair[id] = 0
	}
	for _, a := range allocs {
		users, weights := a.Users, a.Weights
		if len(users) == 0 {
			users = memberIDs
			weights = make([]int64, len(memberIDs))
			for i := range weights {
				weights[i] = 1
			}
		}
		shares := weightedShares(a.Amount, weights)
		for i, u := range users {
			fair[u] += shares[i]
		}
	}
	return fair
}

// computePlan returns a minimal-ish set of transfers that brings every net
// balance to zero. Optimal minimum-transfers is NP-hard, so this is a greedy
// heuristic: repeatedly settle the largest debtor against the largest creditor.
// Ties broken by user id for determinism.
func computePlan(balances []types.MemberBalance) []types.Transfer {
	type acct struct {
		user string
		net  int64
	}
	var debtors, creditors []acct
	for _, b := range balances {
		switch {
		case b.Net < 0:
			debtors = append(debtors, acct{b.UserID, b.Net})
		case b.Net > 0:
			creditors = append(creditors, acct{b.UserID, b.Net})
		}
	}
	sort.Slice(debtors, func(i, j int) bool {
		if debtors[i].net != debtors[j].net {
			return debtors[i].net < debtors[j].net
		}
		return debtors[i].user < debtors[j].user
	})
	sort.Slice(creditors, func(i, j int) bool {
		if creditors[i].net != creditors[j].net {
			return creditors[i].net > creditors[j].net
		}
		return creditors[i].user < creditors[j].user
	})

	plan := []types.Transfer{}
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		owe := -debtors[i].net
		credit := creditors[j].net
		amt := owe
		if credit < amt {
			amt = credit
		}
		plan = append(plan, types.Transfer{From: debtors[i].user, To: creditors[j].user, Amount: amt})
		debtors[i].net += amt
		creditors[j].net -= amt
		if debtors[i].net == 0 {
			i++
		}
		if creditors[j].net == 0 {
			j++
		}
	}
	return plan
}
