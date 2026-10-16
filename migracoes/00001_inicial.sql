-- +goose Up
CREATE TABLE IF NOT EXISTS mensagens (
    chat          TEXT    NOT NULL,
    id            TEXT    NOT NULL,
    remetente     TEXT    NOT NULL,
    remetente_alt TEXT    NOT NULL,
    texto         TEXT    NOT NULL,
    enviada_em    INTEGER NOT NULL,
    PRIMARY KEY (chat, id)
);
CREATE INDEX IF NOT EXISTS mensagens_por_remetente ON mensagens (chat, remetente);
CREATE INDEX IF NOT EXISTS mensagens_por_remetente_alt ON mensagens (chat, remetente_alt);
CREATE INDEX IF NOT EXISTS mensagens_por_data ON mensagens (chat, enviada_em);
CREATE INDEX IF NOT EXISTS mensagens_por_idade ON mensagens (enviada_em);

CREATE TABLE IF NOT EXISTS apelidos (
    chat    TEXT NOT NULL,
    usuario TEXT NOT NULL,
    apelido TEXT NOT NULL,
    PRIMARY KEY (chat, usuario)
);

CREATE TABLE IF NOT EXISTS casamentos (
    chat       TEXT    NOT NULL,
    conjuge1   TEXT    NOT NULL,
    conjuge2   TEXT    NOT NULL,
    casados_em INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS casamentos_por_chat ON casamentos (chat);

CREATE TABLE IF NOT EXISTS adocoes (
    chat       TEXT    NOT NULL,
    pai        TEXT    NOT NULL,
    filho      TEXT    NOT NULL,
    adotado_em INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS adocoes_por_chat ON adocoes (chat);

-- +goose Down
DROP TABLE IF EXISTS adocoes;
DROP TABLE IF EXISTS casamentos;
DROP TABLE IF EXISTS apelidos;
DROP TABLE IF EXISTS mensagens;
