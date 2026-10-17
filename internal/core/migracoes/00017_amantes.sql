-- +goose Up
CREATE TABLE relacionamentos_novos (
    chat    TEXT    NOT NULL,
    tipo    TEXT    NOT NULL CHECK (tipo IN ('namoro', 'casamento', 'amante')),
    pessoa1 TEXT    NOT NULL,
    pessoa2 TEXT    NOT NULL,
    desde   INTEGER NOT NULL
);
INSERT INTO relacionamentos_novos (rowid, chat, tipo, pessoa1, pessoa2, desde)
    SELECT rowid, chat, tipo, pessoa1, pessoa2, desde FROM relacionamentos;
DROP TABLE relacionamentos;
ALTER TABLE relacionamentos_novos RENAME TO relacionamentos;
CREATE INDEX IF NOT EXISTS relacionamentos_por_chat ON relacionamentos (chat, tipo);

-- +goose Down
CREATE TABLE relacionamentos_antigos (
    chat    TEXT    NOT NULL,
    tipo    TEXT    NOT NULL CHECK (tipo IN ('namoro', 'casamento')),
    pessoa1 TEXT    NOT NULL,
    pessoa2 TEXT    NOT NULL,
    desde   INTEGER NOT NULL
);
INSERT INTO relacionamentos_antigos (rowid, chat, tipo, pessoa1, pessoa2, desde)
    SELECT rowid, chat, tipo, pessoa1, pessoa2, desde FROM relacionamentos WHERE tipo != 'amante';
DROP TABLE relacionamentos;
ALTER TABLE relacionamentos_antigos RENAME TO relacionamentos;
CREATE INDEX IF NOT EXISTS relacionamentos_por_chat ON relacionamentos (chat, tipo);
