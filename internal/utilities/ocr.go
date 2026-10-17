package utilities

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	ocrLanguages = "por+eng"
	ocrMaxRunes  = 3500
	ocrMinSize   = "2000x2000<"
)

func (b *bot) ocr(ctx context.Context, evt *events.Message) error {
	media := ocrSource(evt.Message)
	if media == nil {
		return b.ReplyTo(ctx, evt, "🔍 Manda uma foto com !ocr na legenda, ou responde uma foto ou figurinha com !ocr, que eu escrevo o texto que tiver nela")
	}

	var text string
	err := b.WithMediaJob(ctx, evt, func(ctx context.Context) (err error) {
		text, err = b.readImageText(ctx, media)
		return err
	})
	if errors.Is(err, errMediaTooLarge) {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Essa imagem passa de %d MB, é grande demais pra eu ler 😵", mediaMaxMB))
	}
	if err != nil {
		b.Client.Log.Warnf("!ocr: %v", err)
		return b.ReplyTo(ctx, evt, "Não consegui ler essa imagem 😕")
	}
	if text == "" {
		return b.ReplyTo(ctx, evt, "🔍 Não achei nenhum texto nessa imagem")
	}
	return b.ReplyTo(ctx, evt, "📝 *Texto da imagem:*\n\n"+core.TruncateRunes(text, ocrMaxRunes))
}

func ocrSource(msg *waE2E.Message) sizedMedia {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	for _, m := range []*waE2E.Message{msg, quoted} {
		if image := m.GetImageMessage(); image != nil {
			return image
		}
		if sticker := m.GetStickerMessage(); sticker != nil {
			return sticker
		}
	}
	return nil
}

func (b *bot) readImageText(ctx context.Context, media sizedMedia) (text string, err error) {
	if media.GetFileLength() > mediaMaxBytes {
		return "", errMediaTooLarge
	}
	data, err := b.Client.Download(ctx, media)
	if err != nil {
		return "", fmt.Errorf("baixar imagem: %w", err)
	}

	dir, err := os.MkdirTemp("", "ocr-*")
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	in, image, out := filepath.Join(dir, "entrada"), filepath.Join(dir, "imagem.png"), filepath.Join(dir, "texto")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return "", err
	}
	err = core.RunNiceTool(ctx, "magick", in+"[0]", "-background", "white", "-alpha", "remove", "-alpha", "off",
		"-colorspace", "Gray", "-resize", ocrMinSize, image)
	if err != nil {
		return "", err
	}
	if err := core.RunNiceTool(ctx, "tesseract", image, out, "-l", ocrLanguages); err != nil {
		return "", err
	}
	raw, err := os.ReadFile(out + ".txt")
	if err != nil {
		return "", err
	}
	return cleanOCRText(string(raw)), nil
}

func cleanOCRText(raw string) string {
	var lines []string
	blank := true
	for line := range strings.Lines(raw) {
		line = strings.TrimSpace(line)
		if line == "" && blank {
			continue
		}
		blank = line == ""
		lines = append(lines, line)
	}
	text := strings.TrimSpace(strings.Join(lines, "\n"))
	if !strings.ContainsFunc(text, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
		return ""
	}
	return text
}
