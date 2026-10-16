-- +goose Up
ALTER TABLE presidencias ADD COLUMN impeachment_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN impeachment_em;
