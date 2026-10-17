-- +goose Up
CREATE TABLE IF NOT EXISTS guardados (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    qualidade  INTEGER NOT NULL DEFAULT 0,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    usos       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, item, qualidade)
);

-- +goose Down
DROP TABLE IF EXISTS guardados;
