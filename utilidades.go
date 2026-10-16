package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	mediaJobTimeout = 3 * time.Minute
	toolOutputLimit = 500
)

var mediaJobs = make(chan struct{}, 2)

func (b *bot) withMediaJob(ctx context.Context, evt *events.Message, job func(ctx context.Context) error) error {
	b.react(ctx, evt, "⏳")
	defer b.react(context.WithoutCancel(ctx), evt, "")
	select {
	case mediaJobs <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-mediaJobs }()
	ctx, cancel := context.WithTimeout(ctx, mediaJobTimeout)
	defer cancel()
	return job(ctx)
}

func (b *bot) react(ctx context.Context, evt *events.Message, emoji string) {
	chat := evt.Info.Chat
	if _, err := b.client.SendMessage(ctx, chat, b.client.BuildReaction(chat, evt.Info.Sender, evt.Info.ID, emoji)); err != nil {
		b.client.Log.Warnf("reagir com %q: %v", emoji, err)
	}
}

func replyTo(evt *events.Message) *waE2E.ContextInfo {
	return &waE2E.ContextInfo{
		StanzaID:      proto.String(evt.Info.ID),
		Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
		QuotedMessage: evt.Message,
	}
}

func runTool(ctx context.Context, name string, args ...string) error {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out[max(0, len(out)-toolOutputLimit):])))
}
