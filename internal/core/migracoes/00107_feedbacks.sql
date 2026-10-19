-- +goose Up
CREATE TABLE IF NOT EXISTS feedbacks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat TEXT NOT NULL,
    autor TEXT NOT NULL,
    autor_alt TEXT NOT NULL,
    texto TEXT NOT NULL,
    criado_em INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS feedbacks_chat ON feedbacks (chat, id);

-- +goose Down
DROP TABLE IF EXISTS feedbacks;
