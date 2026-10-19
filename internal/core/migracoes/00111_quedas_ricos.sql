-- +goose Up
CREATE TABLE IF NOT EXISTS ricos_ranking (
    chat TEXT NOT NULL,
    pessoa TEXT NOT NULL,
    jid TEXT NOT NULL,
    posicao INTEGER NOT NULL,
    saldo INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

CREATE TABLE IF NOT EXISTS prejuizos (
    chat TEXT NOT NULL,
    pessoa TEXT NOT NULL,
    motivo TEXT NOT NULL,
    valor INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa, motivo)
);

-- +goose Down
DROP TABLE IF EXISTS prejuizos;
DROP TABLE IF EXISTS ricos_ranking;
