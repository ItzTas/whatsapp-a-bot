-- +goose Up
ALTER TABLE presidencias ADD COLUMN ditadura INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN revolta INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN exercito_ate INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN exercito_ate;
ALTER TABLE presidencias DROP COLUMN revolta;
ALTER TABLE presidencias DROP COLUMN ditadura;
