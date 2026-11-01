-- +goose Up
CREATE TABLE carteiras (
    chat                TEXT   NOT NULL,
    pessoa              TEXT   NOT NULL,
    jid                 TEXT   NOT NULL,
    saldo               BIGINT NOT NULL DEFAULT 0 CHECK (saldo >= 0),
    daily_em            BIGINT NOT NULL DEFAULT 0,
    trabalho_em         BIGINT NOT NULL DEFAULT 0,
    roubo_em            BIGINT NOT NULL DEFAULT 0,
    profissao           TEXT   NOT NULL DEFAULT '',
    profissao_em        BIGINT NOT NULL DEFAULT 0,
    pesca_em            BIGINT NOT NULL DEFAULT 0,
    mineracao_em        BIGINT NOT NULL DEFAULT 0,
    lixao_em            BIGINT NOT NULL DEFAULT 0,
    furto_em            BIGINT NOT NULL DEFAULT 0,
    cafeina_em          BIGINT NOT NULL DEFAULT 0,
    prisao_em           BIGINT NOT NULL DEFAULT 0,
    preso_ate           BIGINT NOT NULL DEFAULT 0,
    crime_em            BIGINT NOT NULL DEFAULT 0,
    processo_em         BIGINT NOT NULL DEFAULT 0,
    fuga_em             BIGINT NOT NULL DEFAULT 0,
    sequestro_em        BIGINT NOT NULL DEFAULT 0,
    assalto_em          BIGINT NOT NULL DEFAULT 0,
    denuncia_em         BIGINT NOT NULL DEFAULT 0,
    morto_ate           BIGINT NOT NULL DEFAULT 0,
    assassinato_em      BIGINT NOT NULL DEFAULT 0,
    abuso_em            BIGINT NOT NULL DEFAULT 0,
    libertacao_em       BIGINT NOT NULL DEFAULT 0,
    abuso_trabalho_em   BIGINT NOT NULL DEFAULT 0,
    abuso_roubo_em      BIGINT NOT NULL DEFAULT 0,
    abuso_sequestro_em  BIGINT NOT NULL DEFAULT 0,
    ressurreicao_em     BIGINT NOT NULL DEFAULT 0,
    imovel              BIGINT NOT NULL DEFAULT 0,
    titulo              TEXT   NOT NULL DEFAULT '',
    droga               TEXT   NOT NULL DEFAULT '',
    droga_ate           BIGINT NOT NULL DEFAULT 0,
    encanto_em          BIGINT NOT NULL DEFAULT 0,
    inseto_em           BIGINT NOT NULL DEFAULT 0,
    disfarce_em         BIGINT NOT NULL DEFAULT 0,
    saude               BIGINT NOT NULL DEFAULT 100,
    saude_em            BIGINT NOT NULL DEFAULT 0,
    igreja              BIGINT NOT NULL DEFAULT 0,
    igreja_desde        BIGINT NOT NULL DEFAULT 0,
    conversao_em        BIGINT NOT NULL DEFAULT 0,
    desvio_em           BIGINT NOT NULL DEFAULT 0,
    cofre_casa          BIGINT NOT NULL DEFAULT 0,
    postagem_em         BIGINT NOT NULL DEFAULT 0,
    publi_em            BIGINT NOT NULL DEFAULT 0,
    cratera_em          BIGINT NOT NULL DEFAULT 0,
    canteiros_comprados BIGINT NOT NULL DEFAULT 0,
    escudo_ate          BIGINT NOT NULL DEFAULT 0,
    ficha_em            BIGINT NOT NULL DEFAULT 0,
    aposta_serie        BIGINT NOT NULL DEFAULT 0,
    aposta_lucro        BIGINT NOT NULL DEFAULT 0,
    premio_aposta       BIGINT NOT NULL DEFAULT 0,
    premio_aposta_em    BIGINT NOT NULL DEFAULT 0,
    aposta_perda        BIGINT NOT NULL DEFAULT 0,
    arremesso_em        BIGINT NOT NULL DEFAULT 0,
    alvo_nivel          BIGINT NOT NULL DEFAULT 0,
    alvo_apostado       BIGINT NOT NULL DEFAULT 0,
    prestigio           BIGINT NOT NULL DEFAULT 0,
    premio_viciado      BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa)
);

CREATE INDEX carteiras_por_saldo ON carteiras (chat, saldo DESC);

CREATE TABLE zoeiras (
    chat  TEXT   NOT NULL,
    alvo  TEXT   NOT NULL,
    tipo  TEXT   NOT NULL,
    autor TEXT   NOT NULL,
    ate   BIGINT NOT NULL,
    PRIMARY KEY (chat, alvo, tipo)
);

CREATE TABLE economia_vinculos (
    chat     TEXT   PRIMARY KEY,
    economia TEXT   NOT NULL
);

CREATE TABLE missoes (
    chat      TEXT   NOT NULL,
    pessoa    TEXT   NOT NULL,
    dia       TEXT   NOT NULL,
    missao    TEXT   NOT NULL,
    progresso BIGINT NOT NULL DEFAULT 0,
    meta      BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa, dia, missao)
);

CREATE TABLE fama (
    chat       TEXT   NOT NULL,
    pessoa     TEXT   NOT NULL,
    plataforma TEXT   NOT NULL,
    seguidores BIGINT NOT NULL DEFAULT 0,
    postado_em BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, plataforma)
);

CREATE TABLE eventos_vistos (
    chat     TEXT   NOT NULL,
    pessoa   TEXT   NOT NULL,
    evento   TEXT   NOT NULL,
    vezes    BIGINT NOT NULL DEFAULT 0,
    visto_em BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa, evento)
);

CREATE TABLE eventos_marcados (
    chat     TEXT   NOT NULL,
    pessoa   TEXT   NOT NULL,
    evento   TEXT   NOT NULL,
    a_partir BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa, evento)
);

CREATE TABLE ricos_ranking (
    chat    TEXT   NOT NULL,
    pessoa  TEXT   NOT NULL,
    jid     TEXT   NOT NULL,
    posicao BIGINT NOT NULL,
    saldo   BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

CREATE TABLE prejuizos (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    motivo TEXT   NOT NULL,
    valor  BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa, motivo)
);

CREATE TABLE apostas_lucro (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    dia    TEXT   NOT NULL,
    lucro  BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, dia)
);

CREATE TABLE apostas_recentes (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    em     BIGINT NOT NULL,
    lucro  BIGINT NOT NULL
);

CREATE INDEX apostas_recentes_pessoa ON apostas_recentes (chat, pessoa, em);

CREATE TABLE carreiras (
    chat      TEXT   NOT NULL,
    pessoa    TEXT   NOT NULL,
    profissao TEXT   NOT NULL,
    minutos   BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, profissao)
);

CREATE TABLE apostas_volume (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    em     BIGINT NOT NULL,
    valor  BIGINT NOT NULL
);

CREATE INDEX apostas_volume_pessoa ON apostas_volume (chat, pessoa, em);

CREATE TABLE patrimonios (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    valor  BIGINT NOT NULL,
    em     BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa, em)
);

CREATE TABLE numeros (
    chat   TEXT   NOT NULL,
    pessoa TEXT   NOT NULL,
    numero BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa),
    UNIQUE (chat, numero)
);

CREATE TABLE lendas (
    chat        TEXT   NOT NULL,
    evento      TEXT   NOT NULL,
    pessoa      TEXT   NOT NULL,
    acordada_em BIGINT NOT NULL,
    PRIMARY KEY (chat, evento)
);

-- +goose StatementBegin
CREATE FUNCTION numero_na_carteira() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    PERFORM pg_advisory_xact_lock(hashtext('numeros:' || NEW.chat));
    INSERT INTO numeros (chat, pessoa, numero)
    VALUES (NEW.chat, NEW.pessoa, (SELECT COALESCE(MAX(numero), 0) + 1 FROM numeros WHERE chat = NEW.chat))
    ON CONFLICT DO NOTHING;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER numero_na_carteira AFTER INSERT ON carteiras
FOR EACH ROW EXECUTE FUNCTION numero_na_carteira();

-- +goose Down
DROP TRIGGER IF EXISTS numero_na_carteira ON carteiras;
DROP FUNCTION IF EXISTS numero_na_carteira();
DROP TABLE IF EXISTS lendas;
DROP TABLE IF EXISTS numeros;
DROP TABLE IF EXISTS patrimonios;
DROP TABLE IF EXISTS apostas_volume;
DROP TABLE IF EXISTS carreiras;
DROP TABLE IF EXISTS apostas_recentes;
DROP TABLE IF EXISTS apostas_lucro;
DROP TABLE IF EXISTS prejuizos;
DROP TABLE IF EXISTS ricos_ranking;
DROP TABLE IF EXISTS eventos_marcados;
DROP TABLE IF EXISTS eventos_vistos;
DROP TABLE IF EXISTS fama;
DROP TABLE IF EXISTS missoes;
DROP TABLE IF EXISTS economia_vinculos;
DROP TABLE IF EXISTS zoeiras;
DROP TABLE IF EXISTS carteiras;
