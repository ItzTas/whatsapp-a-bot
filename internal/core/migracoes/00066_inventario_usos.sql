-- +goose Up
ALTER TABLE inventarios ADD COLUMN usos INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE inventarios DROP COLUMN usos;
