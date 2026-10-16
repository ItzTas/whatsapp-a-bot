-- +goose Up
CREATE TABLE IF NOT EXISTS namoros (
    chat            TEXT    NOT NULL,
    namorado1       TEXT    NOT NULL,
    namorado2       TEXT    NOT NULL,
    namorando_desde INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS namoros_por_chat ON namoros (chat);

-- +goose Down
DROP TABLE IF EXISTS namoros;
