-- +goose Up
CREATE TABLE IF NOT EXISTS acoes_precos (
    chat          TEXT    NOT NULL,
    empresa       TEXT    NOT NULL,
    centavos      INTEGER NOT NULL CHECK (centavos > 0),
    abertura      INTEGER NOT NULL CHECK (abertura > 0),
    dia           TEXT    NOT NULL,
    atualizado_em INTEGER NOT NULL,
    PRIMARY KEY (chat, empresa)
);

CREATE TABLE IF NOT EXISTS acoes_posicoes (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    empresa    TEXT    NOT NULL,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    custo      INTEGER NOT NULL CHECK (custo >= 0),
    PRIMARY KEY (chat, pessoa, empresa)
);

-- +goose Down
DROP TABLE IF EXISTS acoes_posicoes;
DROP TABLE IF EXISTS acoes_precos;
