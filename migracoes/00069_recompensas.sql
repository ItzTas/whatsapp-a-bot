-- +goose Up
CREATE TABLE IF NOT EXISTS recompensas (
    chat  TEXT    NOT NULL,
    alvo  TEXT    NOT NULL,
    autor TEXT    NOT NULL,
    valor INTEGER NOT NULL,
    em    INTEGER NOT NULL,
    PRIMARY KEY (chat, alvo, autor)
);

-- +goose Down
DROP TABLE IF EXISTS recompensas;
