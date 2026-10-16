-- +goose Up
ALTER TABLE carteiras ADD COLUMN assalto_em INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN assalto_em;
