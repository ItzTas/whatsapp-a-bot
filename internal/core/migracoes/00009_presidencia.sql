-- +goose Up
CREATE TABLE presidencias (
    chat                         TEXT   PRIMARY KEY,
    pessoa                       TEXT   NOT NULL,
    jid                          TEXT   NOT NULL,
    votos                        BIGINT NOT NULL,
    eleito_em                    BIGINT NOT NULL,
    mandato_ate                  BIGINT NOT NULL,
    abusos                       BIGINT NOT NULL DEFAULT 0,
    imposto_cobrado              BIGINT NOT NULL DEFAULT 0,
    trabalho_pct                 BIGINT NOT NULL DEFAULT 0,
    abusos_trabalho              BIGINT NOT NULL DEFAULT 0,
    abusos_roubo                 BIGINT NOT NULL DEFAULT 0,
    abusos_sequestro             BIGINT NOT NULL DEFAULT 0,
    salario_pct                  BIGINT NOT NULL DEFAULT 0,
    impeachment_em               BIGINT NOT NULL DEFAULT 0,
    daily_pct                    BIGINT NOT NULL DEFAULT 0,
    juros_pct                    BIGINT NOT NULL DEFAULT 0,
    loja_pct                     BIGINT NOT NULL DEFAULT 0,
    pix_pct                      BIGINT NOT NULL DEFAULT 0,
    pena_pct                     BIGINT NOT NULL DEFAULT 0,
    fianca_pct                   BIGINT NOT NULL DEFAULT 0,
    extracao_pct                 BIGINT NOT NULL DEFAULT 0,
    toque_recolher               BIGINT NOT NULL DEFAULT 0,
    auxilio_feito                BIGINT NOT NULL DEFAULT 0,
    anistia_feita                BIGINT NOT NULL DEFAULT 0,
    indultos                     BIGINT NOT NULL DEFAULT 0,
    confisco_feito               BIGINT NOT NULL DEFAULT 0,
    intervencao_feita            BIGINT NOT NULL DEFAULT 0,
    renda_pct                    BIGINT NOT NULL DEFAULT 0,
    poupanca_imposto_pct         BIGINT NOT NULL DEFAULT 0,
    caixa_feita                  BIGINT NOT NULL DEFAULT 0,
    renunciou                    BIGINT NOT NULL DEFAULT 0,
    golpe_em                     BIGINT NOT NULL DEFAULT 0,
    golpista                     BIGINT NOT NULL DEFAULT 0,
    fama_pct                     BIGINT NOT NULL DEFAULT 0,
    sorteado                     BIGINT NOT NULL DEFAULT 0,
    ditadura                     BIGINT NOT NULL DEFAULT 0,
    revolta                      BIGINT NOT NULL DEFAULT 0,
    exercito_ate                 BIGINT NOT NULL DEFAULT 0,
    cassino_pct                  BIGINT NOT NULL DEFAULT 0,
    trabalho_pct_ditador         BIGINT NOT NULL DEFAULT 0,
    salario_pct_ditador          BIGINT NOT NULL DEFAULT 0,
    daily_pct_ditador            BIGINT NOT NULL DEFAULT 0,
    juros_pct_ditador            BIGINT NOT NULL DEFAULT 0,
    poupanca_imposto_pct_ditador BIGINT NOT NULL DEFAULT 0,
    renda_pct_ditador            BIGINT NOT NULL DEFAULT 0,
    loja_pct_ditador             BIGINT NOT NULL DEFAULT 0,
    pix_pct_ditador              BIGINT NOT NULL DEFAULT 0,
    pena_pct_ditador             BIGINT NOT NULL DEFAULT 0,
    fianca_pct_ditador           BIGINT NOT NULL DEFAULT 0,
    extracao_pct_ditador         BIGINT NOT NULL DEFAULT 0,
    fama_pct_ditador             BIGINT NOT NULL DEFAULT 0,
    cassino_pct_ditador          BIGINT NOT NULL DEFAULT 0,
    agitacao                     BIGINT NOT NULL DEFAULT 0,
    venda_pct                    BIGINT NOT NULL DEFAULT 0,
    venda_pct_ditador            BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE presidencia_vagas (
    chat       TEXT   PRIMARY KEY,
    sorteio_em BIGINT NOT NULL
);

CREATE TABLE ditadura_favoritos (
    chat        TEXT   PRIMARY KEY,
    pessoa      TEXT   NOT NULL,
    jid         TEXT   NOT NULL,
    favorito_em BIGINT NOT NULL
);

CREATE TABLE ditadores (
    chat        TEXT   NOT NULL,
    pessoa      TEXT   NOT NULL,
    ditadura_em BIGINT NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

-- +goose Down
DROP TABLE IF EXISTS ditadores;
DROP TABLE IF EXISTS ditadura_favoritos;
DROP TABLE IF EXISTS presidencia_vagas;
DROP TABLE IF EXISTS presidencias;
