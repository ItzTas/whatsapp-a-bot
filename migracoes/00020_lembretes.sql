-- +goose Up
CREATE TABLE IF NOT EXISTS lembretes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    chat        TEXT    NOT NULL,
    hora        INTEGER NOT NULL CHECK (hora BETWEEN 0 AND 23),
    minuto      INTEGER NOT NULL CHECK (minuto BETWEEN 0 AND 59),
    texto       TEXT    NOT NULL CHECK (texto <> ''),
    mencoes     TEXT    NOT NULL DEFAULT '',
    criador     TEXT    NOT NULL,
    criador_alt TEXT    NOT NULL,
    criado_em   INTEGER NOT NULL,
    proximo     INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS lembretes_por_chat ON lembretes (chat);
CREATE INDEX IF NOT EXISTS lembretes_por_proximo ON lembretes (proximo);

-- +goose Down
DROP TABLE IF EXISTS lembretes;
