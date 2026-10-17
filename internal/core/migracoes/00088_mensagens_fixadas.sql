-- +goose Up
CREATE TABLE IF NOT EXISTS mensagens_fixadas (
    chat       TEXT    NOT NULL,
    mensagem   TEXT    NOT NULL,
    remetente  TEXT    NOT NULL,
    ate        INTEGER NOT NULL DEFAULT 0,
    renovar_em INTEGER NOT NULL,
    PRIMARY KEY (chat, mensagem)
);
CREATE INDEX IF NOT EXISTS mensagens_fixadas_por_renovacao ON mensagens_fixadas (renovar_em);

-- +goose Down
DROP TABLE IF EXISTS mensagens_fixadas;
