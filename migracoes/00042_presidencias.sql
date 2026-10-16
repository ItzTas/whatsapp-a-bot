-- +goose Up
CREATE TABLE IF NOT EXISTS presidencias (
    chat            TEXT    PRIMARY KEY,
    pessoa          TEXT    NOT NULL,
    jid             TEXT    NOT NULL,
    votos           INTEGER NOT NULL,
    eleito_em       INTEGER NOT NULL,
    mandato_ate     INTEGER NOT NULL,
    abusos          INTEGER NOT NULL DEFAULT 0,
    imposto_cobrado INTEGER NOT NULL DEFAULT 0,
    trabalho_pct    INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS presidencias;
