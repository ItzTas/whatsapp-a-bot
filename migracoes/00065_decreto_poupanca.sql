-- +goose Up
ALTER TABLE presidencias ADD COLUMN poupanca_imposto_pct INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN poupanca_imposto_pct;
