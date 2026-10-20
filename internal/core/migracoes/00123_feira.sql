-- +goose Up
CREATE TABLE IF NOT EXISTS feira_anuncios (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    chat         TEXT    NOT NULL,
    vendedor     TEXT    NOT NULL,
    vendedor_jid TEXT    NOT NULL,
    item         TEXT    NOT NULL,
    quantidade   INTEGER NOT NULL CHECK (quantidade > 0),
    preco        INTEGER NOT NULL CHECK (preco > 0),
    expira_em    INTEGER NOT NULL,
    UNIQUE (chat, vendedor, item)
);

CREATE INDEX IF NOT EXISTS feira_anuncios_chat ON feira_anuncios (chat, expira_em);

-- +goose Down
DROP TABLE IF EXISTS feira_anuncios;
