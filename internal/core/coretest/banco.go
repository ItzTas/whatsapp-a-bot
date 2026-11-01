package coretest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

const templateDatabase = "modelo"

var server struct {
	url       string
	admin     *sql.DB
	databases atomic.Int64
}

func Run(m *testing.M) int {
	stop, err := start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "subir o postgres dos testes:", err)
		return 1
	}
	code := m.Run()
	if err := stop(); err != nil {
		fmt.Fprintln(os.Stderr, "derrubar o postgres dos testes:", err)
	}
	return code
}

func Database(t testing.TB) string {
	t.Helper()
	name := fmt.Sprintf("teste_%d", server.databases.Add(1))
	if _, err := server.admin.ExecContext(t.Context(), "CREATE DATABASE "+name+" TEMPLATE "+templateDatabase); err != nil {
		t.Fatalf("criar banco de teste: %v", err)
	}
	t.Cleanup(func() {
		if _, err := server.admin.ExecContext(context.Background(), "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
			t.Errorf("apagar banco de teste: %v", err)
		}
	})
	address, err := withDatabase(server.url, name)
	if err != nil {
		t.Fatalf("montar endereço do banco de teste: %v", err)
	}
	return address
}

func MessageStore(t testing.TB) *core.MessageStore {
	t.Helper()
	store, err := core.OpenMessageStore(t.Context(), Database(t))
	if err != nil {
		t.Fatalf("abrir banco de teste: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("fechar banco de teste: %v", err)
		}
	})
	return store
}

func start() (func() error, error) {
	if base := os.Getenv("TEST_DATABASE_URL"); base != "" {
		return useServer(base, func() error { return nil })
	}
	dir, stopCluster, err := startCluster()
	if err != nil {
		return nil, err
	}
	stop, err := useServer("host="+dir+" user=postgres dbname=postgres", stopCluster)
	if err != nil {
		return nil, errors.Join(err, stopCluster())
	}
	return stop, nil
}

func useServer(base string, stopServer func() error) (func() error, error) {
	admin, err := sql.Open("pgx", base)
	if err != nil {
		return nil, err
	}
	server.url, server.admin = base, admin
	if err := waitReady(admin); err != nil {
		return nil, errors.Join(err, admin.Close())
	}
	if err := prepareTemplate(); err != nil {
		return nil, errors.Join(err, admin.Close())
	}
	return func() error {
		_, err := admin.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+templateDatabase+" WITH (FORCE)")
		return errors.Join(err, admin.Close(), stopServer())
	}, nil
}

func prepareTemplate() error {
	ctx := context.Background()
	if _, err := server.admin.ExecContext(ctx, "DROP DATABASE IF EXISTS "+templateDatabase+" WITH (FORCE)"); err != nil {
		return err
	}
	if _, err := server.admin.ExecContext(ctx, "CREATE DATABASE "+templateDatabase); err != nil {
		return err
	}
	address, err := withDatabase(server.url, templateDatabase)
	if err != nil {
		return err
	}
	store, err := core.OpenMessageStore(ctx, address)
	if err != nil {
		return fmt.Errorf("migrar o banco modelo: %w", err)
	}
	if err := store.Close(); err != nil {
		return err
	}
	container, err := sqlstore.New(ctx, "pgx", address, waLog.Noop)
	if err != nil {
		return fmt.Errorf("criar as tabelas do whatsmeow: %w", err)
	}
	return container.Close()
}

func startCluster() (string, func() error, error) {
	dir, err := os.MkdirTemp("", "pg")
	if err != nil {
		return "", nil, err
	}
	data := filepath.Join(dir, "dados")
	initdb := exec.Command("initdb", "-D", data, "-U", "postgres", "-A", "trust", "-E", "UTF8", "--locale=C.UTF-8", "--no-sync")
	if out, err := initdb.CombinedOutput(); err != nil {
		return "", nil, errors.Join(fmt.Errorf("initdb (instale o postgres ou defina TEST_DATABASE_URL): %w\n%s", err, out), os.RemoveAll(dir))
	}
	postgres := exec.Command("postgres", "-D", data, "-k", dir,
		"-c", "listen_addresses=", "-c", "fsync=off", "-c", "synchronous_commit=off", "-c", "full_page_writes=off")
	postgres.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGQUIT}
	exited, err := startTied(postgres)
	if err != nil {
		return "", nil, errors.Join(err, os.RemoveAll(dir))
	}
	return dir, func() error {
		return errors.Join(stopProcess(postgres, exited), os.RemoveAll(dir))
	}, nil
}

func startTied(cmd *exec.Cmd) (<-chan struct{}, error) {
	started := make(chan error, 1)
	exited := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		if err := cmd.Start(); err != nil {
			started <- err
			return
		}
		started <- nil
		_ = cmd.Wait()
		close(exited)
	}()
	return exited, <-started
}

func stopProcess(cmd *exec.Cmd, exited <-chan struct{}) error {
	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		return err
	}
	select {
	case <-exited:
		return nil
	case <-time.After(10 * time.Second):
		return cmd.Process.Kill()
	}
}

func waitReady(admin *sql.DB) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		err := admin.PingContext(context.Background())
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func withDatabase(base, name string) (string, error) {
	if !strings.HasPrefix(base, "postgres://") && !strings.HasPrefix(base, "postgresql://") {
		return base + " dbname=" + name, nil
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	u.Path = "/" + name
	return u.String(), nil
}
