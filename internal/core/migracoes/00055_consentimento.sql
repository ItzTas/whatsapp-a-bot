-- +goose Up
ALTER TABLE relacionamentos ADD COLUMN consentido INTEGER NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS contas_conjuntas (
    chat    TEXT    NOT NULL,
    pessoa1 TEXT    NOT NULL,
    pessoa2 TEXT    NOT NULL,
    saldo   INTEGER NOT NULL DEFAULT 0 CHECK (saldo >= 0),
    PRIMARY KEY (chat, pessoa1, pessoa2)
);

-- +goose Down
DROP TABLE IF EXISTS contas_conjuntas;
ALTER TABLE relacionamentos DROP COLUMN consentido;
