-- +goose Up
CREATE TABLE festas (
    chat      TEXT   NOT NULL PRIMARY KEY,
    ate       BIGINT NOT NULL,
    anfitriao TEXT   NOT NULL,
    total     BIGINT NOT NULL DEFAULT 0
);

ALTER TABLE carteiras ADD COLUMN encomenda_em BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE carteiras DROP COLUMN IF EXISTS encomenda_em;
DROP TABLE IF EXISTS festas;
