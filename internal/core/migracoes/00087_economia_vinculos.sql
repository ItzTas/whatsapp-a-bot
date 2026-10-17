-- +goose Up
CREATE TABLE IF NOT EXISTS economia_vinculos (
    chat     TEXT PRIMARY KEY,
    economia TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS economia_vinculos;
