-- +goose Up
ALTER TABLE carteiras ADD COLUMN inseto_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN inseto_em;
