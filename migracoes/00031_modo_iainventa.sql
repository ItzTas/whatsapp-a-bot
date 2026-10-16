-- +goose Up
INSERT OR IGNORE INTO conversa_modos (chat, modo)
SELECT chat, 'iainventa' FROM conversa_modos WHERE modo = 'iaenxerida';

-- +goose Down
DELETE FROM conversa_modos WHERE modo = 'iainventa';
