-- +goose Up
CREATE TABLE IF NOT EXISTS titulos (
    chat   TEXT    NOT NULL,
    titulo TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    jid    TEXT    NOT NULL,
    pago   INTEGER NOT NULL,
    em     INTEGER NOT NULL,
    PRIMARY KEY (chat, titulo)
);
ALTER TABLE carteiras ADD COLUMN titulo TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE carteiras DROP COLUMN titulo;
DROP TABLE IF EXISTS titulos;
