-- +goose Up
CREATE TABLE IF NOT EXISTS empregados (
    id          INTEGER PRIMARY KEY,
    chat        TEXT    NOT NULL,
    pessoa      TEXT    NOT NULL,
    tipo        TEXT    NOT NULL,
    coletado_em INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS empregados_dono ON empregados (chat, pessoa);

-- +goose Down
DROP INDEX IF EXISTS empregados_dono;
DROP TABLE IF EXISTS empregados;
