-- +goose Up
ALTER TABLE presidencias ADD COLUMN daily_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN juros_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN loja_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN pix_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN pena_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN fianca_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN extracao_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN cassino_fechado INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN toque_recolher INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN auxilio_feito INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN anistia_feita INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN indultos INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN confisco_feito INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN intervencao_feita INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN intervencao_feita;
ALTER TABLE presidencias DROP COLUMN confisco_feito;
ALTER TABLE presidencias DROP COLUMN indultos;
ALTER TABLE presidencias DROP COLUMN anistia_feita;
ALTER TABLE presidencias DROP COLUMN auxilio_feito;
ALTER TABLE presidencias DROP COLUMN toque_recolher;
ALTER TABLE presidencias DROP COLUMN cassino_fechado;
ALTER TABLE presidencias DROP COLUMN extracao_pct;
ALTER TABLE presidencias DROP COLUMN fianca_pct;
ALTER TABLE presidencias DROP COLUMN pena_pct;
ALTER TABLE presidencias DROP COLUMN pix_pct;
ALTER TABLE presidencias DROP COLUMN loja_pct;
ALTER TABLE presidencias DROP COLUMN juros_pct;
ALTER TABLE presidencias DROP COLUMN daily_pct;
