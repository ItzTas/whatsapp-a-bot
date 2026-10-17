-- +goose Up
ALTER TABLE carteiras ADD COLUMN disfarce_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN disfarce_em;
