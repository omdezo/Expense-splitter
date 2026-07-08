-- name: CreateExpense :one
INSERT INTO expenses (group_id, paid_by, amount_baisa, category, description, occurred_on, split_type)
VALUES (@group_id::uuid, @paid_by::uuid, @amount_baisa, @category, @description, @occurred_on::date, @split_type)
RETURNING id, created_at;

-- name: CreateExpenseShare :exec
INSERT INTO expense_shares (expense_id, user_id, weight)
VALUES (@expense_id::uuid, @user_id::uuid, @weight);

-- name: ListExpenseAllocations :many
SELECT e.id, e.amount_baisa, s.user_id, s.weight
FROM expenses e
LEFT JOIN expense_shares s ON s.expense_id = e.id
WHERE e.group_id = @group_id::uuid AND e.deleted_at IS NULL
ORDER BY e.created_at, e.id, s.user_id;

-- name: UserHasExpenseShares :one
SELECT EXISTS(
  SELECT 1
  FROM expense_shares s
  JOIN expenses e ON e.id = s.expense_id
  WHERE e.group_id = @group_id::uuid AND s.user_id = @user_id::uuid AND e.deleted_at IS NULL
) AS has_shares;

-- name: ListExpenses :many
SELECT e.id, m.user_id AS paid_by, e.amount_baisa, e.category, e.description,
       e.occurred_on::text AS occurred_on, e.split_type, e.created_at,
       COUNT(*) OVER()::bigint AS full_count
FROM expenses e
JOIN memberships m ON m.id = e.paid_by
WHERE e.group_id = @group_id::uuid
  AND e.deleted_at IS NULL
  AND (sqlc.narg('category')::expense_category IS NULL OR e.category = sqlc.narg('category')::expense_category)
  AND (sqlc.narg('paid_by')::uuid IS NULL OR m.user_id = sqlc.narg('paid_by')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR e.description ILIKE ('%' || sqlc.narg('search')::text || '%') ESCAPE '\')
ORDER BY e.occurred_on, e.created_at
LIMIT @page_limit::int OFFSET @page_offset::int;

-- name: LockExpenseForUpdate :one
SELECT e.amount_baisa, m.user_id
FROM expenses e
JOIN memberships m ON m.id = e.paid_by
WHERE e.id = @id::uuid AND e.group_id = @group_id::uuid AND e.deleted_at IS NULL
FOR UPDATE OF e;

-- name: UpdateExpense :one
UPDATE expenses
SET amount_baisa = @amount_baisa, category = @category, description = @description,
    occurred_on = @occurred_on::date, updated_at = now()
WHERE id = @id::uuid
RETURNING created_at;

-- name: SoftDeleteExpense :exec
UPDATE expenses
SET deleted_at = now(), updated_at = now()
WHERE id = @id::uuid;

-- name: SumPaidByMember :many
SELECT paid_by, SUM(amount_baisa)::bigint AS total
FROM expenses
WHERE group_id = @group_id::uuid AND deleted_at IS NULL
GROUP BY paid_by;

-- name: SumSpendByCategory :many
SELECT category, SUM(amount_baisa)::bigint AS total
FROM expenses
WHERE group_id = @group_id::uuid AND deleted_at IS NULL
GROUP BY category;
