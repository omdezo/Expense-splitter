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

-- name: SeedGlobalAdmin :execrows
INSERT INTO users (email, is_global_admin, verification_status)
VALUES (@email, true, 'verified')
ON CONFLICT (email) DO NOTHING;
