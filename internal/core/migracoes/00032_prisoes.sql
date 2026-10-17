-- +goose Up
ALTER TABLE carteiras ADD COLUMN prisao_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN preso_ate INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN preso_ate;
ALTER TABLE carteiras DROP COLUMN prisao_em;
