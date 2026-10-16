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

const animatedStickerMaxBytes = 500 << 10

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
	return b.sendText(ctx, chat, "Não consegui fazer a figurinha 😕")
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
		return encodeSticker(ctx, in, out, stickerFrame("512"), "-frames:v", "1", "-quality", "80")
	}
	attempts := []struct{ fps, seconds, quality, size string }{
		{"15", "8", "60", "512"},
		{"15", "8", "35", "512"},
		{"12", "6", "30", "512"},
		{"10", "5", "20", "448"},
		{"8", "4", "10", "384"},
		{"6", "3", "5", "320"},
	}
	for _, a := range attempts {
		filter := fmt.Sprintf("fps=%s,%s", a.fps, stickerFrame(a.size))
		sticker, err := encodeSticker(ctx, in, out, filter, "-t", a.seconds, "-loop", "0", "-an", "-quality", a.quality)
		if err != nil || len(sticker) <= animatedStickerMaxBytes {
			return sticker, err
		}
	}
	return nil, errStickerTooLarge
}

func stickerFrame(size string) string {
	return fmt.Sprintf("scale=%[1]s:%[1]s:force_original_aspect_ratio=decrease,format=rgba,pad=512:512:(ow-iw)/2:(oh-ih)/2:color=black@0", size)
}

func encodeSticker(ctx context.Context, in, out, filter string, extra ...string) ([]byte, error) {
	args := slices.Concat([]string{"-y", "-loglevel", "error", "-i", in, "-vf", filter, "-c:v", "libwebp"}, extra, []string{out})
	if err := runFFmpeg(ctx, args...); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}

func (b *bot) baixarFigurinha(ctx context.Context, evt *events.Message, sticker *waE2E.StickerMessage) error {
	chat := evt.Info.Chat
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error {
		data, err := b.client.Download(ctx, sticker)
		if err != nil {
			return fmt.Errorf("baixar figurinha: %w", err)
		}
		m, err := stickerMedia(ctx, data)
		if err != nil {
			return err
		}
		return b.sendMedia(ctx, chat, m, replyTo(evt))
	})
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("baixar figurinha: %v", err)
	return b.sendText(ctx, chat, "Não consegui transformar essa figurinha em imagem 😕")
}

func stickerMedia(ctx context.Context, sticker []byte) (m outgoing, err error) {
	dir, err := os.MkdirTemp("", "baixar-figurinha-*")
	if err != nil {
		return outgoing{}, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	in := filepath.Join(dir, "figurinha.webp")
	if err := os.WriteFile(in, sticker, 0o600); err != nil {
		return outgoing{}, err
	}
	if isAnimatedWebP(sticker) {
		return animatedStickerMedia(ctx, in)
	}
	out := filepath.Join(dir, "figurinha.png")
	if err := runFFmpeg(ctx, "-y", "-loglevel", "error", "-i", in, "-frames:v", "1", out); err != nil {
		return outgoing{}, err
	}
	return imageMedia(ctx, out)
}

func animatedStickerMedia(ctx context.Context, in string) (outgoing, error) {
	gif := in + ".gif"
	if err := runTool(ctx, "magick", in, "-coalesce", "-background", "white", "-alpha", "remove", "-alpha", "off", gif); err != nil {
		return outgoing{}, err
	}
	return gifMedia(ctx, gif)
}

func isAnimatedWebP(webp []byte) bool {
	return len(webp) > 20 && string(webp[12:16]) == "VP8X" && webp[20]&0x02 != 0
}
