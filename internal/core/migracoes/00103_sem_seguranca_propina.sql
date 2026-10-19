-- +goose Up
ALTER TABLE carteiras DROP COLUMN seguranca_ate;
ALTER TABLE carteiras DROP COLUMN propina_ate;

-- +goose Down
ALTER TABLE carteiras ADD COLUMN seguranca_ate INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN propina_ate INTEGER NOT NULL DEFAULT 0;
