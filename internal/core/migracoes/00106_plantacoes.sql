-- +goose Up
CREATE TABLE IF NOT EXISTS plantacoes (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    chat      TEXT    NOT NULL,
    pessoa    TEXT    NOT NULL,
    semente   TEXT    NOT NULL,
    pronta_em INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS plantacoes_por_pessoa ON plantacoes (chat, pessoa, pronta_em);

ALTER TABLE carteiras ADD COLUMN canteiros_comprados INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN canteiros_comprados;
DROP TABLE IF EXISTS plantacoes;
