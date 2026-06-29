-- +goose Up
CREATE UNIQUE INDEX one_group_admin_per_group ON memberships (group_id) WHERE role = 'group_admin';

-- +goose Down
DROP INDEX one_group_admin_per_group;
