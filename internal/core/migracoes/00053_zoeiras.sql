-- +goose Up
CREATE TABLE IF NOT EXISTS zoeiras (
    chat  TEXT    NOT NULL,
    alvo  TEXT    NOT NULL,
    tipo  TEXT    NOT NULL,
    autor TEXT    NOT NULL,
    ate   INTEGER NOT NULL,
    PRIMARY KEY (chat, alvo, tipo)
);

-- +goose Down
DROP TABLE IF EXISTS zoeiras;
