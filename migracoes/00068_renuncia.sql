-- +goose Up
ALTER TABLE presidencias ADD COLUMN renunciou INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN renunciou;
