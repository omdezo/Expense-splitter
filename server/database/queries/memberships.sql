-- name: GetMembership :one
SELECT id, role, status
FROM memberships
WHERE group_id = @group_id::uuid AND user_id = @user_id::uuid;

-- name: CreateGroupAdminMembership :exec
INSERT INTO memberships (group_id, user_id, role, status)
VALUES (@group_id::uuid, @user_id::uuid, 'group_admin', 'approved');

-- name: CreateJoinRequest :one
INSERT INTO memberships (group_id, user_id)
VALUES (@group_id::uuid, @user_id::uuid)
ON CONFLICT (group_id, user_id) DO NOTHING
RETURNING role, status, created_at;

-- name: DecideJoinRequest :one
UPDATE memberships
SET status = @status, updated_at = now()
WHERE group_id = @group_id::uuid AND user_id = @user_id::uuid AND status = 'requested'
RETURNING role, status, created_at;

-- name: ListJoinRequests :many
SELECT m.user_id, u.email, m.role, m.status, m.created_at
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.group_id = @group_id::uuid AND m.status = 'requested'
ORDER BY m.created_at;

-- name: ListGroupMembers :many
SELECT m.user_id, u.email, m.role, m.status, m.created_at
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.group_id = @group_id::uuid
ORDER BY m.role, m.created_at;

-- name: DemoteGroupAdmin :one
UPDATE memberships
SET role = 'member', updated_at = now()
WHERE group_id = @group_id::uuid AND role = 'group_admin'
RETURNING user_id;

-- name: PromoteToGroupAdmin :one
UPDATE memberships
SET role = 'group_admin', updated_at = now()
WHERE group_id = @group_id::uuid AND user_id = @user_id::uuid AND status = 'approved'
RETURNING role, status, created_at;

-- name: MembershipHasExpenses :one
SELECT EXISTS(
  SELECT 1 FROM expenses WHERE group_id = @group_id::uuid AND paid_by = @paid_by::uuid
) AS has_expenses;

-- name: DeleteMembership :exec
DELETE FROM memberships WHERE id = @id::uuid;

-- name: ListUserMemberships :many
SELECT m.group_id, g.name AS group_name, m.role, m.status, m.created_at
FROM memberships m
JOIN groups g ON g.id = m.group_id
WHERE m.user_id = @user_id::uuid
ORDER BY m.created_at;

-- name: DeleteGroupMemberships :exec
DELETE FROM memberships WHERE group_id = @group_id::uuid;

-- name: ListApprovedMembers :many
SELECT m.id, m.user_id, u.email
FROM memberships m
JOIN users u ON u.id = m.user_id
WHERE m.group_id = @group_id::uuid AND m.status = 'approved'
ORDER BY m.user_id;
