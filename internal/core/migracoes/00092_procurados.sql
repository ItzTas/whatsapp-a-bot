-- +goose Up
CREATE TABLE IF NOT EXISTS procurados (
    chat TEXT NOT NULL,
    dia  TEXT NOT NULL,
    PRIMARY KEY (chat, dia)
);

-- +goose Down
DROP TABLE IF EXISTS procurados;
