-- +goose Up
ALTER TABLE empregados ADD COLUMN tensao INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE empregados DROP COLUMN tensao;
