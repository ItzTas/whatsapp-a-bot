-- +goose Up
CREATE TABLE ditadura_favoritos (
    chat TEXT PRIMARY KEY,
    pessoa TEXT NOT NULL,
    jid TEXT NOT NULL,
    favorito_em INTEGER NOT NULL
);

-- +goose Down
DROP TABLE ditadura_favoritos;
