package services

import (
	"sort"

	"expense-splitter/types"
)

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
