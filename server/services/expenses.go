package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

const descriptionMaxLen = 80

// truncateDescription enforces req #11: a listed description is at most 80
// characters; longer ones are cut on a word boundary and suffixed with "..."
// so the last kept word is always complete ("dinner at the ...", never
// "dinner at the har..."). Rune-aware so multibyte text is never split.
func truncateDescription(s string) string {
	const ellipsis = "..."
	if utf8.RuneCountInString(s) <= descriptionMaxLen {
		return s
	}
	runes := []rune(s)
	budget := runes[:descriptionMaxLen-len(ellipsis)]

	lastSpace := -1
	for i := len(budget) - 1; i >= 0; i-- {
		if budget[i] == ' ' {
			lastSpace = i
			break
		}
	}
	if lastSpace < 0 {
		return string(budget) + ellipsis
	}
	head := strings.TrimRight(string(budget[:lastSpace]), " ")
	return head + " " + ellipsis
}

// escapeLike neutralises LIKE/ILIKE wildcards in a user search term so it is
// matched literally. Paired with `ESCAPE '\'`, a query of "50%" finds the text
// "50%" rather than treating % as "match anything".
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func (s *Services) RecordExpense(ctx context.Context, id types.Identity, groupID string, req types.RecordExpenseRequest) (*types.Expense, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("record expense: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("record expense: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	g, err := qtx.LockGroupForExpense(ctx, repo.LockGroupForExpenseParams{OccurredOn: req.OccurredOn, ID: groupID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("record expense: load group", "error", err)
		return nil, types.NewServerError()
	}
	if g.Status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}
	if !g.InRange {
		return nil, types.NewBadRequestError("occurred_on is outside the trip dates")
	}

	// The caller must be an approved member; their membership id is paid_by, so
	// paid_by == caller by construction (no way to record for someone else).
	m, err := qtx.GetMembership(ctx, repo.GetMembershipParams{GroupID: groupID, UserID: caller.UserID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("you are not a member of this group")
	case err != nil:
		s.logger.Errorw("record expense: load membership", "error", err)
		return nil, types.NewServerError()
	}
	if m.Status != types.MembershipApproved {
		return nil, types.NewForbiddenError("your membership is not approved")
	}

	// Non-equal splits: every named participant must be an approved member of
	// this group, otherwise settlement could assign debt to an outsider.
	for _, share := range req.Shares {
		sm, err := qtx.GetMembership(ctx, repo.GetMembershipParams{GroupID: groupID, UserID: share.UserID})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return nil, types.NewBadRequestError("shares[].user_id must be a member of this group")
		case err != nil:
			s.logger.Errorw("record expense: load share membership", "error", err)
			return nil, types.NewServerError()
		}
		if sm.Status != types.MembershipApproved {
			return nil, types.NewBadRequestError("shares[].user_id must be an approved member")
		}
	}

	created, err := qtx.CreateExpense(ctx, repo.CreateExpenseParams{
		GroupID:     groupID,
		PaidBy:      m.ID,
		AmountBaisa: req.AmountBaisa,
		Category:    req.Category,
		Description: req.Description,
		OccurredOn:  req.OccurredOn,
		SplitType:   req.SplitType(),
	})
	if err != nil {
		s.logger.Errorw("record expense: insert", "error", err)
		return nil, types.NewServerError()
	}

	for _, share := range req.Shares {
		if err := qtx.CreateExpenseShare(ctx, repo.CreateExpenseShareParams{
			ExpenseID: created.ID,
			UserID:    share.UserID,
			Weight:    int32(share.Weight),
		}); err != nil {
			s.logger.Errorw("record expense: insert share", "error", err)
			return nil, types.NewServerError()
		}
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     groupID,
		ActorUserID: caller.UserID,
		Action:      "expense.created",
		After:       expenseAudit{ExpenseID: created.ID, AmountBaisa: req.AmountBaisa, Category: req.Category, OccurredOn: req.OccurredOn},
	}); err != nil {
		s.logger.Errorw("record expense: write audit", "error", err)
		return nil, types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("record expense: commit", "error", err)
		return nil, types.NewServerError()
	}

	return &types.Expense{
		ID:          created.ID,
		GroupID:     groupID,
		PaidBy:      caller.UserID,
		AmountBaisa: req.AmountBaisa,
		Category:    req.Category,
		Description: req.Description,
		OccurredOn:  req.OccurredOn,
		SplitType:   req.SplitType(),
		Shares:      req.Shares,
		CreatedAt:   created.CreatedAt,
	}, nil
}

func (s *Services) ListExpenses(ctx context.Context, id types.Identity, groupID string, filter types.ExpenseFilter) ([]types.Expense, types.APIError) {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("list expenses: resolve caller", "error", err)
		return nil, types.NewServerError()
	}
	if apiErr := s.authz.RequireGroupRole(ctx, caller, groupID, types.RoleMember); apiErr != nil {
		return nil, apiErr
	}

	params := repo.ListExpensesParams{GroupID: groupID}
	if filter.Category != "" {
		params.Category = &filter.Category
	}
	if filter.PaidBy != "" {
		params.PaidBy = &filter.PaidBy
	}
	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		params.Search = &escaped
	}

	rows, err := s.q.ListExpenses(ctx, params)
	if err != nil {
		s.logger.Errorw("list expenses: query", "error", err)
		return nil, types.NewServerError()
	}

	out := make([]types.Expense, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.Expense{
			ID:          r.ID,
			GroupID:     groupID,
			PaidBy:      r.PaidBy,
			AmountBaisa: r.AmountBaisa,
			Category:    r.Category,
			Description: truncateDescription(r.Description),
			OccurredOn:  r.OccurredOn,
			SplitType:   r.SplitType,
			CreatedAt:   r.CreatedAt,
		})
	}
	return out, nil
}

type expenseAmountAudit struct {
	AmountBaisa int64 `json:"amount_baisa"`
}

type expenseAudit struct {
	ExpenseID   string                `json:"expense_id"`
	AmountBaisa int64                 `json:"amount_baisa"`
	Category    types.ExpenseCategory `json:"category,omitempty"`
	OccurredOn  string                `json:"occurred_on,omitempty"`
}

// DeleteExpense soft-deletes the caller's own expense while the group is open
// (req #3 member rights); the deletion is written to the audit log (req #16).
func (s *Services) DeleteExpense(ctx context.Context, id types.Identity, groupID, expenseID string) types.APIError {
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("delete expense: resolve caller", "error", err)
		return types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("delete expense: begin tx", "error", err)
		return types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	status, err := qtx.LockGroupStatus(ctx, groupID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("delete expense: lock group", "error", err)
		return types.NewServerError()
	}
	if status != types.GroupOpen {
		return types.NewConflictError("group is not open")
	}

	old, err := qtx.LockExpenseForUpdate(ctx, repo.LockExpenseForUpdateParams{ID: expenseID, GroupID: groupID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return types.NewNotFoundError("expense not found")
	case err != nil:
		s.logger.Errorw("delete expense: load expense", "error", err)
		return types.NewServerError()
	}
	if old.UserID != caller.UserID {
		return types.NewForbiddenError("you can only delete your own expenses")
	}

	if err := qtx.SoftDeleteExpense(ctx, expenseID); err != nil {
		s.logger.Errorw("delete expense: delete", "error", err)
		return types.NewServerError()
	}

	if err := s.writeAudit(ctx, qtx, auditEntry{
		GroupID:     groupID,
		ActorUserID: caller.UserID,
		Action:      "expense.deleted",
		Before:      expenseAudit{ExpenseID: expenseID, AmountBaisa: old.AmountBaisa},
	}); err != nil {
		s.logger.Errorw("delete expense: write audit", "error", err)
		return types.NewServerError()
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("delete expense: commit", "error", err)
		return types.NewServerError()
	}
	return nil
}

func (s *Services) UpdateExpense(ctx context.Context, id types.Identity, groupID, expenseID string, req types.UpdateExpenseRequest) (*types.Expense, types.APIError) {
	if len(req.Shares) > 0 {
		return nil, types.NewBadRequestError("shares cannot be changed after recording — delete and re-record the expense")
	}
	caller, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("account not registered")
	case err != nil:
		s.logger.Errorw("update expense: resolve caller", "error", err)
		return nil, types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("update expense: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	g, err := qtx.LockGroupForExpense(ctx, repo.LockGroupForExpenseParams{OccurredOn: req.OccurredOn, ID: groupID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("group not found")
	case err != nil:
		s.logger.Errorw("update expense: load group", "error", err)
		return nil, types.NewServerError()
	}
	if g.Status != types.GroupOpen {
		return nil, types.NewConflictError("group is not open")
	}
	if !g.InRange {
		return nil, types.NewBadRequestError("occurred_on is outside the trip dates")
	}

	old, err := qtx.LockExpenseForUpdate(ctx, repo.LockExpenseForUpdateParams{ID: expenseID, GroupID: groupID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("expense not found")
	case err != nil:
		s.logger.Errorw("update expense: load expense", "error", err)
		return nil, types.NewServerError()
	}
	if old.UserID != caller.UserID {
		return nil, types.NewForbiddenError("you can only edit your own expenses")
	}

	createdAt, err := qtx.UpdateExpense(ctx, repo.UpdateExpenseParams{
		AmountBaisa: req.AmountBaisa,
		Category:    req.Category,
		Description: req.Description,
		OccurredOn:  req.OccurredOn,
		ID:          expenseID,
	})
	if err != nil {
		s.logger.Errorw("update expense: update", "error", err)
		return nil, types.NewServerError()
	}

	if req.AmountBaisa != old.AmountBaisa {
		if err := s.writeAudit(ctx, qtx, auditEntry{
			GroupID:     groupID,
			ActorUserID: caller.UserID,
			Action:      ActionExpenseAmountChanged,
			Before:      expenseAmountAudit{AmountBaisa: old.AmountBaisa},
			After:       expenseAmountAudit{AmountBaisa: req.AmountBaisa},
		}); err != nil {
			s.logger.Errorw("update expense: write audit", "error", err)
			return nil, types.NewServerError()
		}
	}

	if err := tx.Commit(ctx); err != nil {
		s.logger.Errorw("update expense: commit", "error", err)
		return nil, types.NewServerError()
	}

	return &types.Expense{
		ID:          expenseID,
		GroupID:     groupID,
		PaidBy:      caller.UserID,
		AmountBaisa: req.AmountBaisa,
		Category:    req.Category,
		Description: req.Description,
		OccurredOn:  req.OccurredOn,
		CreatedAt:   createdAt,
	}, nil
}
