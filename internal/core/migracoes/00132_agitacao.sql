-- +goose Up
ALTER TABLE presidencias ADD COLUMN agitacao INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN agitacao;
