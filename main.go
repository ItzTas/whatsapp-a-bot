package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	gifs, err := loadGIFs()
	if err != nil {
		return fmt.Errorf("carregar gifs: %w", err)
	}

	container, err := sqlstore.New(ctx, "sqlite3", "file:sessao.db?_foreign_keys=on", waLog.Stdout("Database", "WARN", true))
	if err != nil {
		return fmt.Errorf("abrir sessao.db: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("ler dispositivo: %w", err)
	}

	messages, err := openMessageStore(ctx, "mensagens.db")
	if err != nil {
		return fmt.Errorf("abrir mensagens.db: %w", err)
	}
	defer func() { err = errors.Join(err, messages.Close()) }()

	nicknames, err := openNicknameStore(ctx, messages.db)
	if err != nil {
		return err
	}
	marriages, err := openMarriageStore(ctx, messages.db)
	if err != nil {
		return err
	}
	adoptions, err := openAdoptionStore(ctx, messages.db)
	if err != nil {
		return err
	}
	assistant, err := newAssistantFromEnv()
	if err != nil {
		return fmt.Errorf("configurar IA: %w", err)
	}

	sessionCtx, endSession := context.WithCancelCause(ctx)
	defer endSession(nil)

	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))
	b := &bot{client: client, gifs: gifs, messages: messages, nicknames: nicknames, marriages: marriages, adoptions: adoptions, assistant: assistant}
	client.AddEventHandler(func(evt any) {
		switch evt := evt.(type) {
		case *events.Message:
			go b.handleMessage(sessionCtx, evt)
		case *events.HistorySync:
			go b.recordHistory(sessionCtx, evt)
		case *events.LoggedOut:
			endSession(fmt.Errorf("sessão encerrada pelo WhatsApp (%s), reinicie para escanear um novo qr code", evt.Reason))
		}
	})

	if err := connect(sessionCtx, client); err != nil {
		return err
	}
	defer client.Disconnect()

	<-sessionCtx.Done()
	if ctx.Err() != nil {
		return nil
	}
	return context.Cause(sessionCtx)
}

func connect(ctx context.Context, client *whatsmeow.Client) error {
	if client.Store.ID != nil {
		return client.Connect()
	}

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("gerar qr code: %w", err)
	}
	if err := client.Connect(); err != nil {
		return err
	}
	for item := range qrChan {
		if item.Event == whatsmeow.QRChannelEventCode {
			qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
			continue
		}
		if item != whatsmeow.QRChannelSuccess {
			return fmt.Errorf("login falhou: %s", item.Event)
		}
	}
	return nil
}
