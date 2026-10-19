-- +goose Up
CREATE TABLE IF NOT EXISTS fama (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    plataforma TEXT    NOT NULL,
    seguidores INTEGER NOT NULL DEFAULT 0,
    postado_em INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, plataforma)
);

ALTER TABLE carteiras ADD COLUMN postagem_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN publi_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN publi_em;
ALTER TABLE carteiras DROP COLUMN postagem_em;
DROP TABLE IF EXISTS fama;
