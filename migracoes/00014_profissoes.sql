-- +goose Up
ALTER TABLE carteiras ADD COLUMN profissao TEXT NOT NULL DEFAULT '';
ALTER TABLE carteiras ADD COLUMN profissao_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN profissao_em;
ALTER TABLE carteiras DROP COLUMN profissao;
