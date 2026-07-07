-- name: CreateAuditEntry :exec
INSERT INTO audit_log (group_id, actor_user_id, action, before, after)
VALUES (@group_id::uuid, @actor_user_id::uuid, @action, @before, @after);

-- name: ListAuditEntries :many
SELECT id, actor_user_id, action, before, after, created_at
FROM audit_log
WHERE group_id = @group_id::uuid
ORDER BY id;
