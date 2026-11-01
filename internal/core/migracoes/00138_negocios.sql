-- +goose Up
CREATE TABLE IF NOT EXISTS negocios (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    tipo       TEXT    NOT NULL,
    nome       TEXT    NOT NULL DEFAULT '',
    nivel      INTEGER NOT NULL DEFAULT 1,
    estoque    INTEGER NOT NULL DEFAULT 0,
    preco      TEXT    NOT NULL DEFAULT 'normal',
    vendido_em INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, tipo)
);

-- +goose Down
DROP TABLE IF EXISTS negocios;
