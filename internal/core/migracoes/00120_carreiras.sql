-- +goose Up
CREATE TABLE IF NOT EXISTS carreiras (
    chat      TEXT    NOT NULL,
    pessoa    TEXT    NOT NULL,
    profissao TEXT    NOT NULL,
    minutos   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, profissao)
);

-- +goose Down
DROP TABLE IF EXISTS carreiras;
