-- +goose Up
ALTER TABLE igrejas ADD COLUMN templo INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN desvio_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN desvio_em;
ALTER TABLE igrejas DROP COLUMN templo;
