-- +goose Up
CREATE TABLE IF NOT EXISTS implantes (
    chat         TEXT    NOT NULL,
    pessoa       TEXT    NOT NULL,
    parte        TEXT    NOT NULL,
    implante     TEXT    NOT NULL,
    instalado_em INTEGER NOT NULL,
    pago_ate     INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, parte)
);

-- +goose Down
DROP TABLE IF EXISTS implantes;
