-- +goose Up
CREATE TABLE IF NOT EXISTS apostas_lucro (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    dia    TEXT    NOT NULL,
    lucro  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, dia)
);

-- +goose Down
DROP TABLE IF EXISTS apostas_lucro;
