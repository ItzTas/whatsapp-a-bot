-- +goose Up
CREATE TABLE IF NOT EXISTS risadas (
    chat          TEXT    NOT NULL,
    id            TEXT    NOT NULL,
    quem_riu      TEXT    NOT NULL,
    alvo          TEXT    NOT NULL,
    alvo_alt      TEXT    NOT NULL,
    mensagem_alvo TEXT    NOT NULL,
    enviada_em    INTEGER NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS risadas_por_data ON risadas (chat, enviada_em);

-- +goose Down
DROP TABLE IF EXISTS risadas;
