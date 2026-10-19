-- +goose Up
ALTER TABLE presidencias DROP COLUMN cassino_pct;

-- +goose Down
ALTER TABLE presidencias ADD COLUMN cassino_pct INTEGER NOT NULL DEFAULT 0;
