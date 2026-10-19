-- +goose Up
ALTER TABLE carteiras ADD COLUMN aposta_perda INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN aposta_perda;
