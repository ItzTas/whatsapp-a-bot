package core

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
)

const (
	panicModule   = "github.com/ItzTas/personal-projects/whatsapp-a-bot/"
	panicCooldown = 10 * time.Minute
)

func (b *Bot) Rescue(ctx context.Context, chat types.JID) {
	r := recover()
	if r == nil {
		return
	}
	place := panicPlace()
	b.Client.Log.Errorf("panic em %s: %v\n%s", place, r, debug.Stack())
	ctx = context.WithoutCancel(ctx)
	chat = chat.ToNonAD()
	if !chat.IsEmpty() && b.freshPanic(place+"|"+chat.String()) {
		b.warnPanicChat(ctx, chat)
	}
	if b.freshPanic(place) {
		b.warnPanicOwners(ctx, chat, place, r)
	}
}

func panicPlace() string {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs)])
	for {
		frame, more := frames.Next()
		if strings.HasPrefix(frame.Function, panicModule) {
			return fmt.Sprintf("%s (%s:%d)", strings.TrimPrefix(frame.Function, panicModule+"internal/"), filepath.Base(frame.File), frame.Line)
		}
		if !more {
			return "lugar desconhecido"
		}
	}
}

func (b *Bot) freshPanic(key string) bool {
	now := time.Now()
	last, loaded := b.panics.Swap(key, now)
	return !loaded || now.Sub(last.(time.Time)) >= panicCooldown
}

func (b *Bot) warnPanicChat(ctx context.Context, chat types.JID) {
	text := "😵‍💫 Eita, deu um bug feio aqui dentro e o que eu tava fazendo foi pro saco"
	if len(b.owners) > 0 {
		text += ". Já avisei o dono"
	}
	if err := b.SendText(ctx, chat, text); err != nil {
		b.Client.Log.Warnf("avisar panic em %s: %v", chat, err)
	}
}

func (b *Bot) warnPanicOwners(ctx context.Context, chat types.JID, place string, cause any) {
	text := fmt.Sprintf("🚨 *Deu panic no bot*%s\n\n📍 %s\n💥 %v\n\n_O stack completo tá no log_", b.panicChatLabel(ctx, chat), place, cause)
	for _, owner := range b.owners {
		jid := types.NewJID(owner, types.DefaultUserServer)
		if err := b.SendText(ctx, jid, text); err != nil {
			b.Client.Log.Warnf("avisar panic pro dono %s: %v", owner, err)
		}
	}
}

func (b *Bot) panicChatLabel(ctx context.Context, chat types.JID) string {
	if chat.IsEmpty() {
		return ""
	}
	if chat.Server != types.GroupServer {
		return " no privado de " + chat.User
	}
	if label := b.PrivateGroupLabel(ctx, chat); label != "" {
		return label
	}
	return " no grupo " + chat.String()
}
