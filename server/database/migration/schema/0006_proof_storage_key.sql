-- +goose Up
ALTER TABLE proofs ADD COLUMN storage_key text;

-- +goose Down
ALTER TABLE proofs DROP COLUMN storage_key;
