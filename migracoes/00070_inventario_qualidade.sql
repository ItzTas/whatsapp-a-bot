-- +goose Up
CREATE TABLE inventarios_novo (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    qualidade  INTEGER NOT NULL DEFAULT 0,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    usos       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, item, qualidade)
);
INSERT INTO inventarios_novo (chat, pessoa, item, quantidade, usos)
SELECT chat, pessoa, item, quantidade, usos FROM inventarios;
DROP TABLE inventarios;
ALTER TABLE inventarios_novo RENAME TO inventarios;

-- +goose Down
CREATE TABLE inventarios_antigo (
    chat       TEXT    NOT NULL,
    pessoa     TEXT    NOT NULL,
    item       TEXT    NOT NULL,
    quantidade INTEGER NOT NULL CHECK (quantidade > 0),
    usos       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (chat, pessoa, item)
);
INSERT INTO inventarios_antigo (chat, pessoa, item, quantidade, usos)
SELECT chat, pessoa, item, SUM(quantidade), MAX(usos) FROM inventarios GROUP BY chat, pessoa, item;
DROP TABLE inventarios;
ALTER TABLE inventarios_antigo RENAME TO inventarios;
