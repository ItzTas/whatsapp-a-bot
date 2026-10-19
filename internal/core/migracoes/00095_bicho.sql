-- +goose Up
CREATE TABLE IF NOT EXISTS bicho_apostas (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    chat    TEXT    NOT NULL,
    pessoa  TEXT    NOT NULL,
    jid     TEXT    NOT NULL,
    sorteio INTEGER NOT NULL,
    tipo    TEXT    NOT NULL,
    palpite INTEGER NOT NULL CHECK (palpite >= 0),
    valor   INTEGER NOT NULL CHECK (valor > 0),
    sorte   INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS bicho_apostas_por_sorteio ON bicho_apostas (sorteio, chat);
CREATE INDEX IF NOT EXISTS bicho_apostas_por_pessoa ON bicho_apostas (chat, pessoa, sorteio);

CREATE TABLE IF NOT EXISTS bicho_resultados (
    sorteio INTEGER PRIMARY KEY,
    milhar  INTEGER NOT NULL CHECK (milhar BETWEEN 0 AND 9999)
);

-- +goose Down
DROP TABLE IF EXISTS bicho_resultados;
DROP TABLE IF EXISTS bicho_apostas;
