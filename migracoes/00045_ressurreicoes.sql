-- +goose Up
ALTER TABLE carteiras ADD COLUMN ressurreicao_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN ressurreicao_em;
