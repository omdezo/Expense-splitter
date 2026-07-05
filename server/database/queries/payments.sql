-- name: ListPayments :many
SELECT id, from_user_id, to_user_id, amount_baisa, status, created_at
FROM payments
WHERE group_id = @group_id::uuid
ORDER BY created_at, id;
