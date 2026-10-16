-- +goose Up
CREATE TABLE IF NOT EXISTS quemfalou_usadas (
    chat     TEXT NOT NULL,
    mensagem TEXT NOT NULL,
    PRIMARY KEY (chat, mensagem)
);

-- +goose Down
DROP TABLE IF EXISTS quemfalou_usadas;
