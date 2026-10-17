-- +goose Up
CREATE TABLE IF NOT EXISTS aniversarios (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    pessoa_alt TEXT    NOT NULL,
    dia        INTEGER NOT NULL CHECK (dia BETWEEN 1 AND 31),
    mes        INTEGER NOT NULL CHECK (mes BETWEEN 1 AND 12),
    ultimo_ano INTEGER NOT NULL DEFAULT 0,
    criado_em  INTEGER NOT NULL,
    UNIQUE (chat, pessoa)
);
CREATE INDEX IF NOT EXISTS aniversarios_por_alt ON aniversarios (chat, pessoa_alt);
CREATE INDEX IF NOT EXISTS aniversarios_por_mes ON aniversarios (mes, ultimo_ano);

-- +goose Down
DROP TABLE IF EXISTS aniversarios;
