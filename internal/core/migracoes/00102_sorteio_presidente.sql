-- +goose Up
ALTER TABLE presidencias ADD COLUMN sorteado INTEGER NOT NULL DEFAULT 0;
CREATE TABLE IF NOT EXISTS presidencia_vagas (
    chat       TEXT    PRIMARY KEY,
    sorteio_em INTEGER NOT NULL
);
UPDATE presidencias SET mandato_ate = min(mandato_ate, eleito_em + 7200);

-- +goose Down
DROP TABLE IF EXISTS presidencia_vagas;
ALTER TABLE presidencias DROP COLUMN sorteado;
