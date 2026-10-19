-- +goose Up
CREATE TABLE IF NOT EXISTS eventos_vistos (
    chat     TEXT    NOT NULL,
    pessoa   TEXT    NOT NULL,
    evento   TEXT    NOT NULL,
    vezes    INTEGER NOT NULL DEFAULT 0,
    visto_em INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, evento)
);

CREATE TABLE IF NOT EXISTS eventos_marcados (
    chat     TEXT    NOT NULL,
    pessoa   TEXT    NOT NULL,
    evento   TEXT    NOT NULL,
    a_partir INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, evento)
);

-- +goose Down
DROP TABLE IF EXISTS eventos_marcados;
DROP TABLE IF EXISTS eventos_vistos;
