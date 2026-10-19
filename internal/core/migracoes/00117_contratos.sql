-- +goose Up
CREATE TABLE IF NOT EXISTS contratos (
    chat        TEXT    NOT NULL,
    pessoa      TEXT    NOT NULL,
    ramo        TEXT    NOT NULL,
    empresa     TEXT    NOT NULL,
    ate         INTEGER NOT NULL,
    reforco_ate INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, ramo)
);

-- +goose Down
DROP TABLE IF EXISTS contratos;
