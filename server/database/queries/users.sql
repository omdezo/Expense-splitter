-- name: GetUserByKeycloakID :one
SELECT id, email, is_global_admin, verification_status
FROM users
WHERE keycloak_id = @keycloak_id::uuid;

-- name: GetUserByID :one
SELECT id, email, is_global_admin, verification_status
FROM users
WHERE id = @id::uuid;

-- name: LinkUserKeycloakID :one
UPDATE users
SET keycloak_id = @keycloak_id::uuid, updated_at = now()
WHERE email = @email AND keycloak_id IS NULL
RETURNING id, email, is_global_admin, verification_status;

-- name: CreateUser :one
INSERT INTO users (keycloak_id, email)
VALUES (@keycloak_id::uuid, @email)
ON CONFLICT (email) DO NOTHING
RETURNING id, email, is_global_admin, verification_status;

-- name: SetUserVerificationPending :one
UPDATE users
SET verification_status = 'pending_verification', updated_at = now()
WHERE id = @id::uuid
RETURNING id, email, is_global_admin, verification_status;

-- name: SetUserVerification :one
UPDATE users
SET verification_status = @status, verified_by = @verified_by::uuid, updated_at = now()
WHERE id = @id::uuid
RETURNING id, email, is_global_admin, verification_status;

-- name: ListUsers :many
SELECT id, email, is_global_admin, verification_status, (keycloak_id IS NOT NULL)::bool AS linked, created_at,
       COUNT(*) OVER()::bigint AS full_count
FROM users
WHERE (sqlc.narg('status')::verification_status IS NULL OR verification_status = sqlc.narg('status')::verification_status)
ORDER BY created_at
LIMIT @page_limit::int OFFSET @page_offset::int;

-- name: GetUserAdminView :one
SELECT id, email, is_global_admin, verification_status, (keycloak_id IS NOT NULL)::bool AS linked, created_at
FROM users
WHERE id = @id::uuid;

-- name: CountUserMemberships :one
SELECT COUNT(*)::bigint FROM memberships WHERE user_id = @user_id::uuid;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = @id::uuid;

-- name: SeedGlobalAdmin :execrows
INSERT INTO users (email, is_global_admin, verification_status)
VALUES (@email, true, 'verified')
ON CONFLICT (email) DO UPDATE
	SET is_global_admin = true, verification_status = 'verified'
	WHERE users.is_global_admin = false OR users.verification_status <> 'verified';
