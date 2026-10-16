-- +goose Up
CREATE TABLE IF NOT EXISTS nomes (
    chat    TEXT NOT NULL,
    usuario TEXT NOT NULL,
    nome    TEXT NOT NULL,
    PRIMARY KEY (chat, usuario)
);

-- +goose Down
DROP TABLE IF EXISTS nomes;
