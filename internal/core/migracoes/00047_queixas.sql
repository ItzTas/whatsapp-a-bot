-- +goose Up
CREATE TABLE IF NOT EXISTS queixas (
    chat    TEXT    NOT NULL,
    vitima  TEXT    NOT NULL,
    acusado TEXT    NOT NULL,
    motivo  TEXT    NOT NULL,
    em      INTEGER NOT NULL,
    PRIMARY KEY (chat, vitima, acusado)
);

-- +goose Down
DROP TABLE IF EXISTS queixas;
