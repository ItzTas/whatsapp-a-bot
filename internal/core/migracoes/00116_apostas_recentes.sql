-- +goose Up
CREATE TABLE IF NOT EXISTS apostas_recentes (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    em     INTEGER NOT NULL,
    lucro  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS apostas_recentes_pessoa ON apostas_recentes (chat, pessoa, em);

-- +goose Down
DROP TABLE IF EXISTS apostas_recentes;
