-- +goose Up
CREATE TABLE IF NOT EXISTS ia_memorias (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    chat      TEXT    NOT NULL,
    texto     TEXT    NOT NULL CHECK (texto <> ''),
    criada_em INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS ia_memorias_por_chat ON ia_memorias (chat);

-- +goose Down
DROP TABLE IF EXISTS ia_memorias;
