-- +goose Up
CREATE TABLE sequestros (
    chat         TEXT   NOT NULL,
    refem        TEXT   NOT NULL,
    refem_jid    TEXT   NOT NULL,
    sequestrador TEXT  ,
    ate          BIGINT NOT NULL,
    PRIMARY KEY (chat, refem)
);

CREATE UNIQUE INDEX sequestros_por_sequestrador ON sequestros (chat, sequestrador);

CREATE TABLE queixas (
    chat    TEXT   NOT NULL,
    vitima  TEXT   NOT NULL,
    acusado TEXT   NOT NULL,
    motivo  TEXT   NOT NULL,
    em      BIGINT NOT NULL,
    PRIMARY KEY (chat, vitima, acusado)
);

CREATE TABLE recompensas (
    chat  TEXT   NOT NULL,
    alvo  TEXT   NOT NULL,
    autor TEXT   NOT NULL,
    valor BIGINT NOT NULL,
    em    BIGINT NOT NULL,
    PRIMARY KEY (chat, alvo, autor)
);

CREATE TABLE procurados (
    chat TEXT   NOT NULL,
    dia  TEXT   NOT NULL,
    PRIMARY KEY (chat, dia)
);

CREATE TABLE contratos (
    chat        TEXT   NOT NULL,
    pessoa      TEXT   NOT NULL,
    ramo        TEXT   NOT NULL,
    empresa     TEXT   NOT NULL,
    ate         BIGINT NOT NULL,
    reforco_ate BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, ramo)
);

-- +goose Down
DROP TABLE IF EXISTS contratos;
DROP TABLE IF EXISTS procurados;
DROP TABLE IF EXISTS recompensas;
DROP TABLE IF EXISTS queixas;
DROP TABLE IF EXISTS sequestros;
