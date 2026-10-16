-- +goose Up
CREATE TABLE IF NOT EXISTS encantos (
    chat  TEXT    NOT NULL,
    alvo  TEXT    NOT NULL,
    tipo  TEXT    NOT NULL,
    autor TEXT    NOT NULL,
    ate   INTEGER NOT NULL,
    PRIMARY KEY (chat, alvo, tipo)
);
ALTER TABLE carteiras ADD COLUMN encanto_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN encanto_em;
DROP TABLE IF EXISTS encantos;
