-- +goose Up
CREATE TABLE IF NOT EXISTS lojas_reposicoes (
    chat       TEXT    PRIMARY KEY,
    reposta_em INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS lojas_reposicoes;
