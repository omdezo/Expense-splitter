-- name: UnsetCurrentProof :exec
UPDATE proofs SET is_current = false
WHERE payment_id = @payment_id::uuid AND is_current;

-- name: CreateProof :one
INSERT INTO proofs (payment_id, proof_type, sha256, byte_size, note, storage_key)
VALUES (@payment_id::uuid, @proof_type, @sha256, @byte_size, @note, @storage_key)
RETURNING id, created_at;

-- name: GetCurrentProof :one
SELECT id, proof_type, sha256, byte_size, note, storage_key, created_at
FROM proofs
WHERE payment_id = @payment_id::uuid AND is_current;
