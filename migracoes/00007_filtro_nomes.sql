-- +goose Up
ALTER TABLE filtro_listas ADD COLUMN nome TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE filtro_listas DROP COLUMN nome;
