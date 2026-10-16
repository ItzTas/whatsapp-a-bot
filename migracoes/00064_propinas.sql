-- +goose Up
ALTER TABLE carteiras ADD COLUMN propina_ate INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN propina_ate;
