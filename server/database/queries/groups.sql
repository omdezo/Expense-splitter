-- name: CreateGroup :one
INSERT INTO groups (name, start_date, end_date, expected_member_count, created_by)
VALUES (@name, @start_date, @end_date, @expected_member_count, @created_by::uuid)
RETURNING id, name, start_date, end_date, status, invite_token, expected_member_count, created_by, created_at;

-- name: GetGroupByID :one
SELECT id, name, start_date, end_date, status, invite_token, status_token, expected_member_count, created_by, created_at
FROM groups
WHERE id = @id::uuid;

-- name: GetGroupStatus :one
SELECT status FROM groups WHERE id = @id::uuid;

-- name: GetGroupByInviteToken :one
SELECT id, status FROM groups WHERE invite_token = @invite_token::uuid;

-- name: UpdateOpenGroup :one
UPDATE groups
SET name = @name, start_date = @start_date, end_date = @end_date,
    expected_member_count = @expected_member_count, updated_at = now()
WHERE id = @id::uuid AND status = 'open'
RETURNING id, name, start_date, end_date, status, invite_token, expected_member_count, created_by, created_at;

-- name: ListGroupsForUser :many
SELECT g.id, g.name, g.start_date, g.end_date, g.status, g.invite_token,
       g.expected_member_count, g.created_by, g.created_at,
       m.role, m.status AS membership_status
FROM groups g
JOIN memberships m ON m.group_id = g.id
WHERE m.user_id = @user_id::uuid AND m.status = 'approved'
ORDER BY g.created_at DESC;

-- name: LockGroup :one
SELECT status FROM groups WHERE id = @id::uuid FOR UPDATE;

-- name: LockGroupStatus :one
SELECT status FROM groups WHERE id = @id::uuid FOR SHARE;

-- name: LockGroupForExpense :one
SELECT status, (@occurred_on::date BETWEEN start_date::date AND end_date::date)::bool AS in_range
FROM groups
WHERE id = @id::uuid
FOR SHARE;

-- name: MarkGroupClosed :exec
UPDATE groups SET status = 'closed', updated_at = now() WHERE id = @id::uuid;

-- name: MarkGroupSettled :exec
UPDATE groups SET status = 'settled', updated_at = now()
WHERE id = @id::uuid AND status = 'closed';

-- name: ListAllGroups :many
SELECT g.id, g.name, g.start_date, g.end_date, g.status, g.created_by, g.created_at,
       COALESCE((SELECT COUNT(*) FROM memberships m WHERE m.group_id = g.id AND m.status = 'approved'), 0)::bigint AS member_count,
       COALESCE((SELECT SUM(e.amount_baisa) FROM expenses e WHERE e.group_id = g.id AND e.deleted_at IS NULL), 0)::bigint AS total_spent
FROM groups g
ORDER BY g.created_at DESC;

-- name: GroupHasHistory :one
SELECT (EXISTS(SELECT 1 FROM expenses WHERE group_id = @id::uuid)
     OR EXISTS(SELECT 1 FROM payments WHERE group_id = @id::uuid)
     OR EXISTS(SELECT 1 FROM settlement_runs WHERE group_id = @id::uuid)
     OR EXISTS(SELECT 1 FROM audit_log WHERE group_id = @id::uuid))::bool AS has_history;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = @id::uuid;

-- name: GetGroupPublicStatus :one
SELECT g.name,
       g.status,
       COALESCE((SELECT SUM(e.amount_baisa) FROM expenses e WHERE e.group_id = g.id AND e.deleted_at IS NULL), 0)::bigint AS total_spent,
       COALESCE((SELECT COUNT(*) FROM memberships m WHERE m.group_id = g.id AND m.status = 'approved'), 0)::bigint AS member_count
FROM groups g
WHERE g.status_token = @status_token::uuid;
