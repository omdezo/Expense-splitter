-- name: CreateGroup :one
INSERT INTO groups (name, start_date, end_date, expected_member_count, created_by)
VALUES (@name, @start_date, @end_date, @expected_member_count, @created_by::uuid)
RETURNING id, name, start_date, end_date, status, invite_token, expected_member_count, created_by, created_at;

-- name: GetGroupByID :one
SELECT id, name, start_date, end_date, status, invite_token, expected_member_count, created_by, created_at
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

-- name: LockGroupForExpense :one
SELECT status, (@occurred_on::date BETWEEN start_date::date AND end_date::date)::bool AS in_range
FROM groups
WHERE id = @id::uuid
FOR SHARE;

-- name: MarkGroupClosed :exec
UPDATE groups SET status = 'closed', updated_at = now() WHERE id = @id::uuid;
