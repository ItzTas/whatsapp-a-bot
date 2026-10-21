-- +goose Up
ALTER TABLE presidencias ADD COLUMN venda_pct INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN venda_pct_ditador INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE presidencias DROP COLUMN venda_pct;
ALTER TABLE presidencias DROP COLUMN venda_pct_ditador;
