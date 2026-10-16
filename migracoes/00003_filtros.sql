-- +goose Up
CREATE TABLE IF NOT EXISTS filtro_modos (
    alvo TEXT PRIMARY KEY,
    modo TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS filtro_listas (
    alvo   TEXT NOT NULL,
    lista  TEXT NOT NULL,
    id     TEXT NOT NULL,
    id_alt TEXT NOT NULL,
    PRIMARY KEY (alvo, lista, id)
);
CREATE INDEX IF NOT EXISTS filtro_listas_por_alt ON filtro_listas (alvo, lista, id_alt);

-- +goose Down
DROP TABLE IF EXISTS filtro_listas;
DROP TABLE IF EXISTS filtro_modos;
