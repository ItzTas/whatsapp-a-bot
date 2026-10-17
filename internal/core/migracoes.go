package core

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed migracoes/*.sql
var migrations embed.FS

func migrate(ctx context.Context, db *sql.DB) error {
	dir, err := fs.Sub(migrations, "migracoes")
	if err != nil {
		return err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, dir)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
