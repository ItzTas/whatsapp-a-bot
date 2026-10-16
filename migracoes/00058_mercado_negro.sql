-- +goose Up
ALTER TABLE carteiras ADD COLUMN droga TEXT NOT NULL DEFAULT '';
ALTER TABLE carteiras ADD COLUMN droga_ate INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN droga_ate;
ALTER TABLE carteiras DROP COLUMN droga;
