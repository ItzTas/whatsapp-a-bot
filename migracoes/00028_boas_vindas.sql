-- +goose Up
CREATE TABLE IF NOT EXISTS boas_vindas (
    chat          TEXT    PRIMARY KEY,
    texto         TEXT    NOT NULL CHECK (texto <> ''),
    autor         TEXT    NOT NULL,
    autor_alt     TEXT    NOT NULL,
    atualizada_em INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS boas_vindas;
