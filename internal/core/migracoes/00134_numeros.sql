-- +goose Up
CREATE TABLE IF NOT EXISTS numeros (
    chat   TEXT    NOT NULL,
    pessoa TEXT    NOT NULL,
    numero INTEGER NOT NULL,
    PRIMARY KEY (chat, pessoa),
    UNIQUE (chat, numero)
);

INSERT INTO numeros (chat, pessoa, numero)
SELECT chat, pessoa, ROW_NUMBER() OVER (PARTITION BY chat ORDER BY rowid)
FROM carteiras;

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS numero_na_carteira AFTER INSERT ON carteiras
BEGIN
    INSERT OR IGNORE INTO numeros (chat, pessoa, numero)
    VALUES (NEW.chat, NEW.pessoa, (SELECT COALESCE(MAX(numero), 0) + 1 FROM numeros WHERE chat = NEW.chat));
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS numero_na_carteira;
DROP TABLE IF EXISTS numeros;
