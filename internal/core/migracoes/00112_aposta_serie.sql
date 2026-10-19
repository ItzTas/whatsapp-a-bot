-- +goose Up
ALTER TABLE carteiras ADD COLUMN aposta_serie INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN aposta_lucro INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN aposta_lucro;
ALTER TABLE carteiras DROP COLUMN aposta_serie;
