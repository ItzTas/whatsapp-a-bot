-- +goose Up
CREATE TABLE IF NOT EXISTS dividas (
    chat         TEXT    NOT NULL,
    devedor      TEXT    NOT NULL,
    devedor_jid  TEXT    NOT NULL,
    credor       TEXT    NOT NULL,
    credor_jid   TEXT    NOT NULL,
    divida       INTEGER NOT NULL CHECK (divida > 0),
    juros        INTEGER NOT NULL CHECK (juros >= 0),
    juros_atraso INTEGER NOT NULL CHECK (juros_atraso >= 0),
    desde        INTEGER NOT NULL,
    vence_em     INTEGER NOT NULL,
    PRIMARY KEY (chat, devedor, credor)
);

CREATE INDEX IF NOT EXISTS dividas_credor ON dividas (chat, credor);

-- +goose Down
DROP TABLE IF EXISTS dividas;
