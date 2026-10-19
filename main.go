package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/ai"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/city"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/economy"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/games"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/jokes"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/llm"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/meters"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/moderation"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/romance"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/stats"
	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/utilities"
)

type module interface {
	Register()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) (err error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("ler .env: %w", err)
	}

	gifs, err := core.LoadGIFs()
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

	messages, err := core.OpenMessageStore(ctx, "mensagens.db")
	if err != nil {
		return fmt.Errorf("abrir mensagens.db: %w", err)
	}
	defer func() { err = errors.Join(err, messages.Close()) }()

	assistant, err := llm.NewAssistantFromEnv()
	if err != nil {
		return fmt.Errorf("configurar IA: %w", err)
	}

	sessionCtx, endSession := context.WithCancelCause(ctx)
	defer endSession(nil)

	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "INFO", true))
	b, err := assemble(ctx, client, gifs, messages, assistant)
	if err != nil {
		return err
	}
	client.AddEventHandler(func(evt any) {
		switch evt := evt.(type) {
		case *events.Message:
			go b.HandleMessage(sessionCtx, evt)
		case *events.GroupInfo:
			go b.HandleGroupInfo(sessionCtx, evt)
		case *events.HistorySync:
			go b.RecordHistory(sessionCtx, evt)
		case *events.LoggedOut:
			endSession(fmt.Errorf("sessão encerrada pelo WhatsApp (%s), reinicie para escanear um novo qr code", evt.Reason))
		}
	})

	b.PrepareUpdate(sessionCtx)
	if err := connect(sessionCtx, client); err != nil {
		return err
	}
	defer client.Disconnect()
	go b.RestoreGames(sessionCtx)
	b.StartJobs(sessionCtx)

	<-sessionCtx.Done()
	if ctx.Err() != nil {
		return nil
	}
	return context.Cause(sessionCtx)
}

func assemble(ctx context.Context, client *whatsmeow.Client, gifs map[string][]core.GIF, messages *core.MessageStore, assistant *llm.Assistant) (*core.Bot, error) {
	b := core.New(client, gifs, messages, assistant)
	econ, err := economy.New(ctx, b)
	if err != nil {
		return nil, err
	}
	rom := romance.New(b, econ)
	gms := games.New(b, econ)
	intelligence := ai.New(b, econ, gms, rom)
	modules := []module{
		rom,
		gms,
		city.New(b),
		meters.New(b, intelligence),
		stats.New(b, econ),
		jokes.New(b, econ, rom),
		econ,
		intelligence,
		moderation.New(b),
		utilities.New(b),
	}
	for _, m := range modules {
		m.Register()
	}
	return b, nil
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
