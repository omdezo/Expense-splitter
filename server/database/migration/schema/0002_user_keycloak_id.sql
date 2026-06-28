-- +goose Up
-- Link a local users row to its Keycloak subject (sub). Nullable so a row can be
-- seeded (e.g. the default global admin) before the subject is known, then
-- claimed on first registration. Postgres allows multiple NULLs under UNIQUE.
ALTER TABLE users ADD COLUMN keycloak_id uuid UNIQUE;

-- +goose Down
ALTER TABLE users DROP COLUMN keycloak_id;
