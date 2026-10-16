-- +goose Up
ALTER TABLE carteiras ADD COLUMN imovel INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN imovel;
