package core

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	mediaJobTimeout = 3 * time.Minute
	toolOutputLimit = 500
	toolNiceness    = "10"
)

var (
	mediaJobs     = make(chan struct{}, 2)
	ffmpegThreads = strconv.Itoa(max(1, runtime.NumCPU()/2))
)

func (b *Bot) WithMediaJob(ctx context.Context, evt *events.Message, job func(ctx context.Context) error) error {
	return b.WithTimedMediaJob(ctx, evt, mediaJobTimeout, job)
}

func (b *Bot) WithTimedMediaJob(ctx context.Context, evt *events.Message, timeout time.Duration, job func(ctx context.Context) error) (err error) {
	b.React(ctx, evt, "⏳")
	defer func() {
		finalReaction := "❌"
		if err == nil {
			finalReaction = "✅"
		}
		b.React(context.WithoutCancel(ctx), evt, finalReaction)
	}()
	select {
	case mediaJobs <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-mediaJobs }()
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return job(ctx)
}

func (b *Bot) React(ctx context.Context, evt *events.Message, emoji string) {
	chat := evt.Info.Chat
	if _, err := b.Client.SendMessage(ctx, chat, b.Client.BuildReaction(chat, evt.Info.Sender, evt.Info.ID, emoji)); err != nil {
		b.Client.Log.Warnf("reagir com %q: %v", emoji, err)
	}
}

func ReplyTo(evt *events.Message) *waE2E.ContextInfo {
	return &waE2E.ContextInfo{
		StanzaID:      proto.String(evt.Info.ID),
		Participant:   proto.String(evt.Info.Sender.ToNonAD().String()),
		QuotedMessage: evt.Message,
	}
}

func RunTool(ctx context.Context, name string, args ...string) error {
	return runCommand(name, exec.CommandContext(ctx, name, args...))
}

func RunNiceTool(ctx context.Context, name string, args ...string) error {
	return runCommand(name, exec.CommandContext(ctx, "nice", slices.Concat([]string{"-n", toolNiceness, name}, args)...))
}

func RunFFmpeg(ctx context.Context, args ...string) error {
	options, output := args[:len(args)-1], args[len(args)-1]
	return RunNiceTool(ctx, "ffmpeg", slices.Concat(
		[]string{"-filter_threads", ffmpegThreads, "-threads", ffmpegThreads},
		options,
		[]string{"-threads", ffmpegThreads, output},
	)...)
}

func runCommand(name string, cmd *exec.Cmd) error {
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out[max(0, len(out)-toolOutputLimit):])))
}
