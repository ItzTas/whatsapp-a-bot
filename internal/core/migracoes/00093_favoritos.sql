-- +goose Up
CREATE TABLE IF NOT EXISTS favoritos (
    chat   TEXT NOT NULL,
    pessoa TEXT NOT NULL,
    item   TEXT NOT NULL,
    PRIMARY KEY (chat, pessoa, item)
);

-- +goose Down
DROP TABLE IF EXISTS favoritos;
