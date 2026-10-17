-- +goose Up
DROP TABLE IF EXISTS emprestimos;

CREATE TABLE emprestimos (
    chat         TEXT    NOT NULL,
    pessoa       TEXT    NOT NULL,
    jid          TEXT    NOT NULL,
    divida       INTEGER NOT NULL CHECK (divida > 0),
    juros        INTEGER NOT NULL CHECK (juros >= 0),
    juros_atraso INTEGER NOT NULL CHECK (juros_atraso >= 0),
    desde        INTEGER NOT NULL,
    vence_em     INTEGER NOT NULL,
    penhora_em   INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

CREATE INDEX IF NOT EXISTS emprestimos_penhora_em ON emprestimos (penhora_em);

-- +goose Down
DROP TABLE IF EXISTS emprestimos;

CREATE TABLE IF NOT EXISTS emprestimos (
    chat      TEXT    NOT NULL,
    pessoa    TEXT    NOT NULL,
    divida    INTEGER NOT NULL CHECK (divida > 0),
    juros     INTEGER NOT NULL CHECK (juros >= 0),
    desde     INTEGER NOT NULL,
    vence_em  INTEGER NOT NULL,
    cobrancas INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa)
);
