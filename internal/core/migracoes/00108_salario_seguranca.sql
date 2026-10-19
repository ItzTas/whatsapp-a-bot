-- +goose Up
UPDATE empregados SET coletado_em = unixepoch() + 86400 WHERE tipo = 'seguranca';

-- +goose Down
UPDATE empregados SET coletado_em = unixepoch() WHERE tipo = 'seguranca';
