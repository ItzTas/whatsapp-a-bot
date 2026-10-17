-- +goose Up
CREATE TABLE IF NOT EXISTS sequestros (
    chat         TEXT    NOT NULL,
    refem        TEXT    NOT NULL,
    refem_jid    TEXT    NOT NULL,
    sequestrador TEXT,
    ate          INTEGER NOT NULL,
    PRIMARY KEY (chat, refem)
);
CREATE UNIQUE INDEX IF NOT EXISTS sequestros_por_sequestrador ON sequestros (chat, sequestrador);
ALTER TABLE carteiras ADD COLUMN sequestro_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN sequestro_em;
DROP TABLE IF EXISTS sequestros;
