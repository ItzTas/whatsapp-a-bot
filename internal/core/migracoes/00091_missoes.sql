-- +goose Up
CREATE TABLE IF NOT EXISTS missoes (
    chat      TEXT    NOT NULL,
    pessoa    TEXT    NOT NULL,
    dia       TEXT    NOT NULL,
    missao    TEXT    NOT NULL,
    progresso INTEGER NOT NULL DEFAULT 0,
    meta      INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, dia, missao)
);

-- +goose Down
DROP TABLE IF EXISTS missoes;
