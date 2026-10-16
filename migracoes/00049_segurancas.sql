-- +goose Up
ALTER TABLE carteiras ADD COLUMN seguranca_ate INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN seguranca_ate;
