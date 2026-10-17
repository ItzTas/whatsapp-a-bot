-- +goose Up
CREATE TABLE IF NOT EXISTS inventarios (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    PRIMARY KEY (chat, pessoa, item)
);
ALTER TABLE carteiras ADD COLUMN pesca_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN mineracao_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN lixao_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN furto_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN furto_em;
ALTER TABLE carteiras DROP COLUMN lixao_em;
ALTER TABLE carteiras DROP COLUMN mineracao_em;
ALTER TABLE carteiras DROP COLUMN pesca_em;
DROP TABLE IF EXISTS inventarios;
