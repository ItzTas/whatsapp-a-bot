-- +goose Up
CREATE TABLE IF NOT EXISTS loteria_bilhetes (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    jid        TEXT    NOT NULL,
    sorteio    INTEGER NOT NULL,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    PRIMARY KEY (chat, pessoa, sorteio)
);
CREATE INDEX IF NOT EXISTS loteria_bilhetes_por_sorteio ON loteria_bilhetes (sorteio);

CREATE TABLE IF NOT EXISTS loteria_sorteios (
    chat          TEXT    NOT NULL,
    sorteio       INTEGER NOT NULL,
    vencedor      TEXT    NOT NULL,
    jid           TEXT    NOT NULL,
    premio        INTEGER NOT NULL CHECK (premio > 0),
    bilhetes      INTEGER NOT NULL CHECK (bilhetes > 0),
    participantes INTEGER NOT NULL CHECK (participantes > 1),
    PRIMARY KEY (chat, sorteio)
);

-- +goose Down
DROP TABLE IF EXISTS loteria_sorteios;
DROP TABLE IF EXISTS loteria_bilhetes;
