-- +goose Up
CREATE TABLE IF NOT EXISTS algemados (
    chat   TEXT NOT NULL,
    id     TEXT NOT NULL,
    id_alt TEXT NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS algemados_por_alt ON algemados (chat, id_alt);

-- +goose Down
DROP TABLE IF EXISTS algemados;
