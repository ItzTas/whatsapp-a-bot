-- +goose Up
CREATE TABLE IF NOT EXISTS carteiras (
    chat        TEXT    NOT NULL,
    pessoa      TEXT    NOT NULL,
    jid         TEXT    NOT NULL,
    saldo       INTEGER NOT NULL DEFAULT 0 CHECK (saldo >= 0),
    daily_em    INTEGER NOT NULL DEFAULT 0,
    trabalho_em INTEGER NOT NULL DEFAULT 0,
    roubo_em    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa)
);
CREATE INDEX IF NOT EXISTS carteiras_por_saldo ON carteiras (chat, saldo DESC);

-- +goose Down
DROP TABLE IF EXISTS carteiras;
