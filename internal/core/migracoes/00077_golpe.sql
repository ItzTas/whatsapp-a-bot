-- +goose Up
ALTER TABLE presidencias ADD COLUMN golpe_em INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN golpista INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN golpista;
ALTER TABLE presidencias DROP COLUMN golpe_em;
