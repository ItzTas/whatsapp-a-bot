-- +goose Up
CREATE TABLE IF NOT EXISTS patrimonios (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    valor  INTEGER NOT NULL,
    em     INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, em)
);

-- +goose Down
DROP TABLE IF EXISTS patrimonios;
