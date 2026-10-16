-- +goose Up
CREATE TABLE IF NOT EXISTS poupancas (
    chat     TEXT    NOT NULL,
    pessoa   TEXT    NOT NULL,
    valor    INTEGER NOT NULL CHECK (valor > 0),
    juros    INTEGER NOT NULL CHECK (juros >= 0),
    vence_em INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

-- +goose Down
DROP TABLE IF EXISTS poupancas;
