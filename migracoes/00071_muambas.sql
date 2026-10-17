-- +goose Up
CREATE TABLE IF NOT EXISTS muambas (
    chat      TEXT NOT NULL,
    dia       TEXT NOT NULL,
    item      TEXT NOT NULL,
    comprador TEXT NOT NULL,
    PRIMARY KEY (chat, dia, item)
);

-- +goose Down
DROP TABLE IF EXISTS muambas;
