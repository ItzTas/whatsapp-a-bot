-- +goose Up
ALTER TABLE carteiras ADD COLUMN escudo_ate INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN ficha_em INTEGER NOT NULL DEFAULT 0;
UPDATE carteiras SET ficha_em = crime_em;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN ficha_em;
ALTER TABLE carteiras DROP COLUMN escudo_ate;
