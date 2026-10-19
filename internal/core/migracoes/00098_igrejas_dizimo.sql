-- +goose Up
ALTER TABLE igrejas ADD COLUMN dizimo INTEGER;

-- +goose Down
ALTER TABLE igrejas DROP COLUMN dizimo;
