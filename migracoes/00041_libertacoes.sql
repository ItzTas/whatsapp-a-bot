-- +goose Up
ALTER TABLE carteiras ADD COLUMN libertacao_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN libertacao_em;
