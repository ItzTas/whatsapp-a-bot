-- +goose Up
ALTER TABLE carteiras ADD COLUMN crime_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN crime_em;
