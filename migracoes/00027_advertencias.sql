-- +goose Up
CREATE TABLE IF NOT EXISTS advertencias (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    pessoa_alt TEXT    NOT NULL,
    motivo     TEXT    NOT NULL DEFAULT '',
    autor      TEXT    NOT NULL,
    autor_alt  TEXT    NOT NULL,
    criada_em  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS advertencias_por_pessoa ON advertencias (chat, pessoa);
CREATE INDEX IF NOT EXISTS advertencias_por_alt ON advertencias (chat, pessoa_alt);

-- +goose Down
DROP TABLE IF EXISTS advertencias;
