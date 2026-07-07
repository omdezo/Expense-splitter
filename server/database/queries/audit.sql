-- name: CreateAuditEntry :exec
INSERT INTO audit_log (group_id, actor_user_id, action, before, after)
VALUES (sqlc.narg('group_id')::uuid, @actor_user_id::uuid, @action, @before, @after);

-- name: ListAuditEntries :many
SELECT id, actor_user_id, action, before, after, created_at
FROM audit_log
WHERE group_id = @group_id::uuid
ORDER BY id;

-- name: ListPaymentAuditActors :many
SELECT actor_user_id, action, (after->>'payment_id')::text AS payment_id
FROM audit_log
WHERE group_id = @group_id::uuid
  AND action LIKE 'payment.%'
  AND after->>'payment_id' IS NOT NULL
ORDER BY id;
