-- +goose Up
ALTER TABLE encantos ADD COLUMN parte INTEGER NOT NULL DEFAULT 0;
ALTER TABLE encantos ADD COLUMN bonus INTEGER NOT NULL DEFAULT 0;
ALTER TABLE encantos ADD COLUMN teto INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS encantos_avisos (
    id        INTEGER PRIMARY KEY,
    chat      TEXT    NOT NULL,
    tipo      TEXT    NOT NULL,
    alvo      TEXT    NOT NULL,
    alvo_jid  TEXT    NOT NULL,
    autor     TEXT    NOT NULL,
    autor_jid TEXT    NOT NULL,
    valor     INTEGER NOT NULL,
    repasse   INTEGER NOT NULL,
    aplicado  INTEGER NOT NULL DEFAULT 0
);

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS encantos_no_ganho AFTER UPDATE OF saldo ON carteiras
WHEN NEW.saldo > OLD.saldo AND EXISTS (
    SELECT 1 FROM encantos e
    WHERE e.chat = NEW.chat AND e.alvo = NEW.pessoa AND (e.parte > 0 OR e.bonus > 0)
      AND e.ate > CAST(strftime('%s', 'now') AS INTEGER)
)
BEGIN
    INSERT INTO encantos_avisos (chat, tipo, alvo, alvo_jid, autor, autor_jid, valor, repasse)
    SELECT NEW.chat, e.tipo, NEW.pessoa, NEW.jid, e.autor, c.jid, (NEW.saldo - OLD.saldo) * e.parte / 100, 1
    FROM encantos e
    JOIN carteiras c ON c.chat = e.chat AND c.pessoa = e.autor
    WHERE e.chat = NEW.chat AND e.alvo = NEW.pessoa AND e.autor <> e.alvo AND e.parte > 0
      AND e.ate > CAST(strftime('%s', 'now') AS INTEGER)
      AND (NEW.saldo - OLD.saldo) * e.parte / 100 > 0;

    INSERT INTO encantos_avisos (chat, tipo, alvo, alvo_jid, autor, autor_jid, valor, repasse)
    SELECT NEW.chat, e.tipo, NEW.pessoa, NEW.jid, e.autor, COALESCE(c.jid, NEW.jid), MIN((NEW.saldo - OLD.saldo) * e.bonus / 100, e.teto), 0
    FROM encantos e
    LEFT JOIN carteiras c ON c.chat = e.chat AND c.pessoa = e.autor
    WHERE e.chat = NEW.chat AND e.alvo = NEW.pessoa AND e.bonus > 0
      AND e.ate > CAST(strftime('%s', 'now') AS INTEGER)
      AND MIN((NEW.saldo - OLD.saldo) * e.bonus / 100, e.teto) > 0;

    UPDATE carteiras SET saldo = saldo + (
        SELECT SUM(a.valor) FROM encantos_avisos a
        WHERE a.chat = NEW.chat AND a.alvo = NEW.pessoa AND a.aplicado = 0 AND a.repasse = 1 AND a.autor = carteiras.pessoa
    )
    WHERE chat = NEW.chat AND pessoa IN (
        SELECT a.autor FROM encantos_avisos a
        WHERE a.chat = NEW.chat AND a.alvo = NEW.pessoa AND a.aplicado = 0 AND a.repasse = 1
    );

    UPDATE carteiras SET saldo = saldo + COALESCE((
        SELECT SUM(CASE WHEN a.repasse = 1 THEN -a.valor ELSE a.valor END) FROM encantos_avisos a
        WHERE a.chat = NEW.chat AND a.alvo = NEW.pessoa AND a.aplicado = 0
    ), 0)
    WHERE chat = NEW.chat AND pessoa = NEW.pessoa;

    DELETE FROM encantos WHERE chat = NEW.chat AND alvo = NEW.pessoa AND tipo IN (
        SELECT a.tipo FROM encantos_avisos a
        WHERE a.chat = NEW.chat AND a.alvo = NEW.pessoa AND a.aplicado = 0
    );

    UPDATE encantos_avisos SET aplicado = 1 WHERE chat = NEW.chat AND alvo = NEW.pessoa AND aplicado = 0;
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS encantos_no_ganho;
DROP TABLE IF EXISTS encantos_avisos;
ALTER TABLE encantos DROP COLUMN teto;
ALTER TABLE encantos DROP COLUMN bonus;
ALTER TABLE encantos DROP COLUMN parte;
