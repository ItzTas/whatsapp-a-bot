-- +goose Up
CREATE TABLE IF NOT EXISTS lendas (
    chat        TEXT    NOT NULL,
    evento      TEXT    NOT NULL,
    pessoa      TEXT    NOT NULL,
    acordada_em INTEGER NOT NULL,
    PRIMARY KEY (chat, evento)
);

-- +goose Down
DROP TABLE IF EXISTS lendas;
