-- +goose Up
CREATE TABLE IF NOT EXISTS perseguidos (
    chat   TEXT NOT NULL,
    id     TEXT NOT NULL,
    id_alt TEXT NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS perseguidos_por_alt ON perseguidos (chat, id_alt);

-- +goose Down
DROP TABLE IF EXISTS perseguidos;
