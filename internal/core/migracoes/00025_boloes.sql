-- +goose Up
CREATE TABLE IF NOT EXISTS boloes (
    chat        TEXT    NOT NULL PRIMARY KEY,
    criador     TEXT    NOT NULL,
    criador_jid TEXT    NOT NULL,
    pergunta    TEXT    NOT NULL,
    fechado     INTEGER NOT NULL DEFAULT 0,
    criado_em   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS bolao_opcoes (
    chat   TEXT    NOT NULL,
    numero INTEGER NOT NULL CHECK (numero > 0),
    texto  TEXT    NOT NULL,
    PRIMARY KEY (chat, numero)
);

CREATE TABLE IF NOT EXISTS bolao_apostas (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    jid    TEXT    NOT NULL,
    opcao  INTEGER NOT NULL,
    valor  INTEGER NOT NULL CHECK (valor > 0),
    PRIMARY KEY (chat, pessoa, opcao)
);

-- +goose Down
DROP TABLE IF EXISTS bolao_apostas;
DROP TABLE IF EXISTS bolao_opcoes;
DROP TABLE IF EXISTS boloes;
