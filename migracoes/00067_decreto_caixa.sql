-- +goose Up
ALTER TABLE presidencias ADD COLUMN caixa_feita INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN caixa_feita;
