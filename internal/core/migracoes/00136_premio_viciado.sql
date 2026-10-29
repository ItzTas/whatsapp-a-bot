-- +goose Up
ALTER TABLE carteiras ADD COLUMN premio_viciado INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN premio_viciado;
