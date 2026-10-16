-- +goose Up
CREATE TABLE IF NOT EXISTS impostos (
    chat  TEXT    NOT NULL,
    dia   TEXT    NOT NULL,
    total INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, dia)
);

-- +goose Down
DROP TABLE IF EXISTS impostos;
