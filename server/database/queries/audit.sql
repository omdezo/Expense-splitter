-- name: CreateAuditEntry :exec
INSERT INTO audit_log (group_id, actor_user_id, action, before, after)
VALUES (@group_id::uuid, @actor_user_id::uuid, @action, @before, @after);
