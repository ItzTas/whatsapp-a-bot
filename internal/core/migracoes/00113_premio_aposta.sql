-- +goose Up
ALTER TABLE carteiras ADD COLUMN premio_aposta INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN premio_aposta_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN premio_aposta_em;
ALTER TABLE carteiras DROP COLUMN premio_aposta;
