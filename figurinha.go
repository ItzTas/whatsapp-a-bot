package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	stickerMaxSeconds       = "8"
	animatedStickerMaxBytes = 500 << 10
	stickerFrame            = "scale=512:512:force_original_aspect_ratio=decrease,format=rgba,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=black@0"
)

var errStickerTooLarge = errors.New("figurinha animada passa de 500 KB mesmo na menor qualidade")

func (b *bot) figurinha(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	source, animated, found := stickerSource(evt.Message)
	if !found {
		return b.sendText(ctx, chat, "Manda uma foto, vídeo ou gif com !figurinha na legenda, ou responde um deles com !figurinha")
	}
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error {
		data, err := b.client.Download(ctx, source)
		if err != nil {
			return fmt.Errorf("baixar mídia: %w", err)
		}
		sticker, err := makeSticker(ctx, data, animated)
		if err != nil {
			return err
		}
		return b.sendSticker(ctx, chat, sticker, replyTo(evt))
	})
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("fazer figurinha: %v", err)
	return b.sendText(ctx, chat, "Não consegui fazer a figurinha 😕 Se for vídeo, tenta um mais curto")
}

func stickerSource(msg *waE2E.Message) (source whatsmeow.DownloadableMessage, animated, found bool) {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	for _, m := range []*waE2E.Message{msg, quoted} {
		if image := m.GetImageMessage(); image != nil {
			return image, false, true
		}
		if video := m.GetVideoMessage(); video != nil {
			return video, true, true
		}
	}
	return nil, false, false
}

func makeSticker(ctx context.Context, data []byte, animated bool) (sticker []byte, err error) {
	dir, err := os.MkdirTemp("", "figurinha-*")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	in, out := filepath.Join(dir, "entrada"), filepath.Join(dir, "figurinha.webp")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, err
	}
	if !animated {
		return encodeSticker(ctx, in, out, stickerFrame, "-frames:v", "1", "-quality", "80")
	}
	for _, quality := range []string{"60", "35", "15"} {
		sticker, err := encodeSticker(ctx, in, out, "fps=15,"+stickerFrame, "-t", stickerMaxSeconds, "-loop", "0", "-an", "-quality", quality)
		if err != nil || len(sticker) <= animatedStickerMaxBytes {
			return sticker, err
		}
	}
	return nil, errStickerTooLarge
}

func encodeSticker(ctx context.Context, in, out, filter string, extra ...string) ([]byte, error) {
	args := slices.Concat([]string{"-y", "-loglevel", "error", "-i", in, "-vf", filter, "-c:v", "libwebp"}, extra, []string{out})
	if err := runTool(ctx, "ffmpeg", args...); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}

func isAnimatedWebP(webp []byte) bool {
	return len(webp) > 20 && string(webp[12:16]) == "VP8X" && webp[20]&0x02 != 0
}
