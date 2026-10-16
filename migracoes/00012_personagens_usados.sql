-- +goose Up
CREATE TABLE IF NOT EXISTS personagens_usados (
    chat       TEXT    NOT NULL,
    personagem INTEGER NOT NULL,
    PRIMARY KEY (chat, personagem)
);

-- +goose Down
DROP TABLE IF EXISTS personagens_usados;
