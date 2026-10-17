-- +goose Up
CREATE TABLE IF NOT EXISTS leiloes (
    chat          TEXT    NOT NULL,
    dia           TEXT    NOT NULL,
    item          TEXT    NOT NULL,
    lance         INTEGER NOT NULL,
    licitante     TEXT    NOT NULL,
    licitante_jid TEXT    NOT NULL,
    fecha_em      INTEGER NOT NULL,
    PRIMARY KEY (chat, dia)
);

CREATE INDEX IF NOT EXISTS leiloes_fecha_em ON leiloes (fecha_em);

-- +goose Down
DROP TABLE IF EXISTS leiloes;
