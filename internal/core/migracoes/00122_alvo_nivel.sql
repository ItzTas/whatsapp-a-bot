-- +goose Up
ALTER TABLE carteiras ADD COLUMN alvo_nivel INTEGER NOT NULL DEFAULT 0;
ALTER TABLE carteiras ADD COLUMN alvo_apostado INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN alvo_apostado;
ALTER TABLE carteiras DROP COLUMN alvo_nivel;
