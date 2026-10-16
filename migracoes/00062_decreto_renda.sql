-- +goose Up
ALTER TABLE presidencias ADD COLUMN renda_pct INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN renda_pct;
