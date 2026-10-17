-- +goose Up
CREATE TABLE IF NOT EXISTS igrejas (
    id         INTEGER PRIMARY KEY,
    chat       TEXT    NOT NULL,
    chave      TEXT    NOT NULL,
    nome       TEXT    NOT NULL,
    doutrina   TEXT    NOT NULL,
    lider      TEXT    NOT NULL,
    cofre      INTEGER NOT NULL DEFAULT 0,
    fundada_em INTEGER NOT NULL,
    UNIQUE (chat, chave)
);
ALTER TABLE carteiras ADD COLUMN igreja INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN igreja_desde INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN conversao_em INTEGER NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS carteiras_por_igreja ON carteiras (chat, igreja);

-- +goose Down
DROP INDEX IF EXISTS carteiras_por_igreja;
ALTER TABLE carteiras DROP COLUMN conversao_em;
ALTER TABLE carteiras DROP COLUMN igreja_desde;
ALTER TABLE carteiras DROP COLUMN igreja;
DROP TABLE IF EXISTS igrejas;
