-- +goose Up
CREATE TABLE IF NOT EXISTS apostas_volume (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    em     INTEGER NOT NULL,
    valor  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS apostas_volume_pessoa ON apostas_volume (chat, pessoa, em);

-- +goose Down
DROP TABLE IF EXISTS apostas_volume;
