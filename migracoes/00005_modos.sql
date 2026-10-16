-- +goose Up
CREATE TABLE IF NOT EXISTS conversa_modos (
    chat TEXT NOT NULL,
    modo TEXT NOT NULL,
    PRIMARY KEY (chat, modo)
);

-- +goose Down
DROP TABLE IF EXISTS conversa_modos;
