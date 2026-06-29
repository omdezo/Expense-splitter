-- +goose Up
ALTER TABLE groups
    ALTER COLUMN start_date TYPE timestamptz USING start_date::timestamptz,
    ALTER COLUMN end_date   TYPE timestamptz USING end_date::timestamptz;

-- +goose Down
ALTER TABLE groups
    ALTER COLUMN start_date TYPE date USING start_date::date,
    ALTER COLUMN end_date   TYPE date USING end_date::date;
