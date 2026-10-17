-- +goose Up
ALTER TABLE carteiras ADD COLUMN processo_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN processo_em;
