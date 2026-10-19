-- +goose Up
CREATE TABLE IF NOT EXISTS versoes (
    revisao TEXT PRIMARY KEY,
    versao TEXT NOT NULL,
    instalada_em INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS versoes;
