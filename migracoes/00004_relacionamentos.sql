-- +goose Up
CREATE TABLE IF NOT EXISTS relacionamentos (
    chat    TEXT    NOT NULL,
    tipo    TEXT    NOT NULL CHECK (tipo IN ('namoro', 'casamento')),
    pessoa1 TEXT    NOT NULL,
    pessoa2 TEXT    NOT NULL,
    desde   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS relacionamentos_por_chat ON relacionamentos (chat, tipo);

-- +goose Down
DROP TABLE IF EXISTS relacionamentos;
