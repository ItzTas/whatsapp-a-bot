-- +goose Up
CREATE TABLE IF NOT EXISTS obras (
    chat       TEXT    NOT NULL PRIMARY KEY,
    nivel      INTEGER NOT NULL DEFAULT 0,
    arrecadado INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS obras_doacoes (
    chat   TEXT    NOT NULL,
    nivel  INTEGER NOT NULL,
    pessoa TEXT    NOT NULL,
    jid    TEXT    NOT NULL,
    valor  INTEGER NOT NULL,
    PRIMARY KEY (chat, nivel, pessoa)
);
CREATE TABLE IF NOT EXISTS obras_concluidas (
    chat        TEXT    NOT NULL,
    nivel       INTEGER NOT NULL,
    homenageado TEXT    NOT NULL,
    em          INTEGER NOT NULL,
    PRIMARY KEY (chat, nivel)
);

-- +goose Down
DROP TABLE IF EXISTS obras_concluidas;
DROP TABLE IF EXISTS obras_doacoes;
DROP TABLE IF EXISTS obras;
