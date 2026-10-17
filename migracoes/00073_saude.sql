-- +goose Up
ALTER TABLE carteiras ADD COLUMN saude INTEGER NOT NULL DEFAULT 100;
ALTER TABLE carteiras ADD COLUMN saude_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN saude_em;
ALTER TABLE carteiras DROP COLUMN saude;
