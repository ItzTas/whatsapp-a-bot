-- +goose Up
ALTER TABLE implantes ADD COLUMN desgaste INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE implantes DROP COLUMN desgaste;
