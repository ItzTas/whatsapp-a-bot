package core

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	maxOpenConns = 20
	lockTimeout  = "5s"
)

type RowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func SQLPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

func BoolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func OpenDatabase(url string) (*sql.DB, error) {
	config, err := pgx.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	config.RuntimeParams["lock_timeout"] = lockTimeout
	db := sql.OpenDB(rebindConnector{stdlib.GetConnector(*config)})
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns / 4)
	db.SetConnMaxIdleTime(5 * time.Minute)
	return db, nil
}

type rebindConnector struct {
	driver.Connector
}

func (c rebindConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &rebindConn{conn.(*stdlib.Conn)}, nil
}

type rebindConn struct {
	*stdlib.Conn
}

func (c *rebindConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(rebind(query))
}

func (c *rebindConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.Conn.PrepareContext(ctx, rebind(query))
}

func (c *rebindConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return c.Conn.ExecContext(ctx, rebind(query), args)
}

func (c *rebindConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.Conn.QueryContext(ctx, rebind(query), args)
}

func rebind(query string) string {
	if !strings.Contains(query, "?") {
		return query
	}
	closers := map[string]string{"'": "'", `"`: `"`, "--": "\n", "/*": "*/"}
	var b strings.Builder
	b.Grow(len(query) + 16)
	n := 0
	for i := 0; i < len(query); {
		if span := quotedSpan(query[i:], closers); span > 0 {
			b.WriteString(query[i : i+span])
			i += span
			continue
		}
		if query[i] != '?' {
			b.WriteByte(query[i])
			i++
			continue
		}
		n++
		b.WriteString("$" + strconv.Itoa(n))
		i++
	}
	return b.String()
}

func quotedSpan(rest string, closers map[string]string) int {
	for open, end := range closers {
		if !strings.HasPrefix(rest, open) {
			continue
		}
		i := strings.Index(rest[len(open):], end)
		if i < 0 {
			return len(rest)
		}
		return len(open) + i + len(end)
	}
	return 0
}
