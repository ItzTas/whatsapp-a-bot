-- +goose Up
CREATE TABLE ditadores (
    chat TEXT NOT NULL,
    pessoa TEXT NOT NULL,
    ditadura_em INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa)
);

-- +goose Down
DROP TABLE ditadores;
