-- +goose Up
CREATE TABLE IF NOT EXISTS reacoes (
    chat   TEXT NOT NULL,
    id     TEXT NOT NULL,
    id_alt TEXT NOT NULL,
    emoji  TEXT NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS reacoes_por_alt ON reacoes (chat, id_alt);

-- +goose Down
DROP TABLE IF EXISTS reacoes;
