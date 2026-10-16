-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS emprestimos;
