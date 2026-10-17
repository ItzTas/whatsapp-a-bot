-- +goose Up
CREATE TABLE IF NOT EXISTS ia_respostas (
    chat        TEXT    NOT NULL,
    id          TEXT    NOT NULL,
    pergunta_id TEXT    NOT NULL,
    texto       TEXT    NOT NULL,
    enviada_em  INTEGER NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS ia_respostas_por_data ON ia_respostas (chat, enviada_em);
CREATE INDEX IF NOT EXISTS ia_respostas_por_pergunta ON ia_respostas (chat, pergunta_id);

-- +goose Down
DROP TABLE IF EXISTS ia_respostas;
