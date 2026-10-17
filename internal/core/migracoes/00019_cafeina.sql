-- +goose Up
ALTER TABLE carteiras ADD COLUMN cafeina_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN cafeina_em;
