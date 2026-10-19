-- +goose Up
ALTER TABLE presidencias ADD COLUMN fama_pct INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN fama_pct;
