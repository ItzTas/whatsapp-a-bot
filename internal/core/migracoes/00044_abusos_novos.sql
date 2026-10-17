-- +goose Up
ALTER TABLE carteiras ADD COLUMN abuso_trabalho_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN abuso_roubo_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN abuso_sequestro_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN abusos_trabalho INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN abusos_roubo INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN abusos_sequestro INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN abusos_sequestro;
ALTER TABLE presidencias DROP COLUMN abusos_roubo;
ALTER TABLE presidencias DROP COLUMN abusos_trabalho;
ALTER TABLE carteiras DROP COLUMN abuso_sequestro_em;
ALTER TABLE carteiras DROP COLUMN abuso_roubo_em;
ALTER TABLE carteiras DROP COLUMN abuso_trabalho_em;
