-- +goose Up
ALTER TABLE carteiras ADD COLUMN cofre_casa INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN cofre_casa;
