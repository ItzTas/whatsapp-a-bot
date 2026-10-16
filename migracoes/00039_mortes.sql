-- +goose Up
ALTER TABLE carteiras ADD COLUMN morto_ate INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN assassinato_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN assassinato_em;
ALTER TABLE carteiras DROP COLUMN morto_ate;
