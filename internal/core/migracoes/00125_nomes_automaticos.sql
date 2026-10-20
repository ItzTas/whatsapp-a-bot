-- +goose Up
ALTER TABLE nomes ADD COLUMN automatico INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE nomes DROP COLUMN automatico;
