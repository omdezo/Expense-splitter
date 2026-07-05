-- name: CreateExpense :one
INSERT INTO expenses (group_id, paid_by, amount_baisa, category, description, occurred_on)
VALUES (@group_id::uuid, @paid_by::uuid, @amount_baisa, @category, @description, @occurred_on::date)
RETURNING id, created_at;

-- name: ListExpenses :many
SELECT e.id, m.user_id AS paid_by, e.amount_baisa, e.category, e.description,
       e.occurred_on::text AS occurred_on, e.created_at
FROM expenses e
JOIN memberships m ON m.id = e.paid_by
WHERE e.group_id = @group_id::uuid
  AND e.deleted_at IS NULL
  AND (sqlc.narg('category')::expense_category IS NULL OR e.category = sqlc.narg('category')::expense_category)
  AND (sqlc.narg('paid_by')::uuid IS NULL OR m.user_id = sqlc.narg('paid_by')::uuid)
  AND (sqlc.narg('search')::text IS NULL OR e.description ILIKE ('%' || sqlc.narg('search')::text || '%') ESCAPE '\')
ORDER BY e.occurred_on, e.created_at;

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
