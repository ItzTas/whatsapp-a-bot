-- +goose Up
ALTER TABLE carteiras ADD COLUMN abuso_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN abuso_em;
