-- name: ListPayments :many
SELECT id, from_user_id, to_user_id, amount_baisa, status, created_at
FROM payments
WHERE group_id = @group_id::uuid
ORDER BY created_at, id;

-- name: GetPayment :one
SELECT id, group_id, from_user_id, to_user_id, amount_baisa, status, version
FROM payments
WHERE id = @id::uuid;

-- name: TransitionPayment :one
UPDATE payments
SET status = @status, version = version + 1, updated_at = now()
WHERE id = @id::uuid AND version = @version
RETURNING id, from_user_id, to_user_id, amount_baisa, status, created_at;

-- name: CountUnsettledPayments :one
SELECT COUNT(*)::bigint AS unsettled
FROM payments
WHERE group_id = @group_id::uuid AND status <> 'settled';
