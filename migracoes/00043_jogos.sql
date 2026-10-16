-- +goose Up
CREATE TABLE IF NOT EXISTS jogos (
    tipo      TEXT    NOT NULL,
    chave     TEXT    NOT NULL,
    estado    TEXT    NOT NULL,
    expira_em INTEGER NOT NULL,
    PRIMARY KEY (tipo, chave)
);

-- +goose Down
DROP TABLE IF EXISTS jogos;
