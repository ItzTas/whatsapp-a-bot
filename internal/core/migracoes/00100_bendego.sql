-- +goose Up
ALTER TABLE carteiras ADD COLUMN cratera_em INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS campo_gravitacional (
    chat      TEXT    PRIMARY KEY,
    acumulado INTEGER NOT NULL DEFAULT 0,
    pago_em   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS montagens_avisos (
    id         INTEGER PRIMARY KEY,
    chat       TEXT    NOT NULL,
    jid        TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    quantidade INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS montagens_avisos;
DROP TABLE IF EXISTS campo_gravitacional;
ALTER TABLE carteiras DROP COLUMN cratera_em;
