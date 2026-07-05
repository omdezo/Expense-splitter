-- +goose Up
ALTER TABLE proofs ADD COLUMN note text;

-- +goose Down
ALTER TABLE proofs DROP COLUMN note;
