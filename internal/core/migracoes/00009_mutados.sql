-- +goose Up
CREATE TABLE IF NOT EXISTS mutados (
    chat   TEXT NOT NULL,
    id     TEXT NOT NULL,
    id_alt TEXT NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS mutados_por_alt ON mutados (chat, id_alt);

-- +goose Down
DROP TABLE IF EXISTS mutados;
