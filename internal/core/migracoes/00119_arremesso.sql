-- +goose Up
ALTER TABLE carteiras ADD COLUMN arremesso_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN arremesso_em;
