-- +goose Up
UPDATE presidencias SET cassino_pct = CASE
    WHEN apostas_pct > 0 THEN -min(apostas_pct, 10)
    WHEN cassino_pct < 0 THEN 0
    ELSE cassino_pct
END;
ALTER TABLE presidencias DROP COLUMN apostas_pct;
ALTER TABLE presidencias DROP COLUMN cassino_fechado;

-- +goose Down
ALTER TABLE presidencias ADD COLUMN cassino_fechado INTEGER NOT NULL DEFAULT 0;
ALTER TABLE presidencias ADD COLUMN apostas_pct INTEGER NOT NULL DEFAULT 0;
UPDATE presidencias SET apostas_pct = -cassino_pct, cassino_pct = 0 WHERE cassino_pct < 0;
