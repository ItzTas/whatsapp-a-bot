-- +goose Up
CREATE TABLE IF NOT EXISTS compras_diarias (
    chat       TEXT    NOT NULL,
    dia        TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    PRIMARY KEY (chat, dia, pessoa, item)
);

-- +goose Down
DROP TABLE IF EXISTS compras_diarias;
