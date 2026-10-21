-- +goose Up
ALTER TABLE presidencias ADD COLUMN trabalho_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN salario_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN daily_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN juros_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN poupanca_imposto_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN renda_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN loja_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN pix_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN pena_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN fianca_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN extracao_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN fama_pct_ditador INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN cassino_pct_ditador INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN trabalho_pct_ditador;
ALTER TABLE presidencias DROP COLUMN salario_pct_ditador;
ALTER TABLE presidencias DROP COLUMN daily_pct_ditador;
ALTER TABLE presidencias DROP COLUMN juros_pct_ditador;
ALTER TABLE presidencias DROP COLUMN poupanca_imposto_pct_ditador;
ALTER TABLE presidencias DROP COLUMN renda_pct_ditador;
ALTER TABLE presidencias DROP COLUMN loja_pct_ditador;
ALTER TABLE presidencias DROP COLUMN pix_pct_ditador;
ALTER TABLE presidencias DROP COLUMN pena_pct_ditador;
ALTER TABLE presidencias DROP COLUMN fianca_pct_ditador;
ALTER TABLE presidencias DROP COLUMN extracao_pct_ditador;
ALTER TABLE presidencias DROP COLUMN fama_pct_ditador;
ALTER TABLE presidencias DROP COLUMN cassino_pct_ditador;
