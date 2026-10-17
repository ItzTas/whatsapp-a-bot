-- +goose Up
CREATE TABLE IF NOT EXISTS completa_usadas (
    chat     TEXT NOT NULL,
    mensagem TEXT NOT NULL,
    PRIMARY KEY (chat, mensagem)
);

-- +goose Down
DROP TABLE IF EXISTS completa_usadas;
