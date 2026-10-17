package utilities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
)

const transcriptionLanguage = "pt"

type transcriber struct {
	endpoint string
	key      string
	model    string
}

func transcriberFromEnv() *transcriber {
	url, model := os.Getenv("STT_URL"), os.Getenv("STT_MODEL")
	if url == "" || model == "" {
		return nil
	}
	return &transcriber{
		endpoint: strings.TrimSuffix(url, "/") + "/audio/transcriptions",
		key:      os.Getenv("STT_KEY"),
		model:    model,
	}
}

func (b *bot) transcrever(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	if b.transcriber == nil {
		return b.SendText(ctx, chat, "🎙️ A transcrição tá desligada, ninguém configurou o STT_URL e o STT_MODEL")
	}
	media := transcriptionMedia(evt.Message)
	if media == nil {
		return b.SendText(ctx, chat, "Responde um áudio ou um vídeo com !transcrever")
	}

	var text string
	err := b.WithMediaJob(ctx, evt, func(ctx context.Context) (err error) {
		text, err = b.transcribeMedia(ctx, media)
		return err
	})
	if err != nil {
		b.Client.Log.Warnf("!transcrever: %v", err)
		return b.ReplyTo(ctx, evt, fmt.Sprintf("Não consegui transcrever 😕 Talvez o arquivo passe de %d MB ou o serviço de transcrição esteja fora do ar", mediaMaxMB))
	}
	if text == "" {
		return b.ReplyTo(ctx, evt, "🤫 Não deu pra entender nada nesse áudio")
	}
	return b.ReplyTo(ctx, evt, "📝 *Transcrição:*\n\n"+text)
}

func transcriptionMedia(msg *waE2E.Message) sizedMedia {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	for _, m := range []*waE2E.Message{msg, quoted} {
		if media := audioMedia(m); media != nil {
			return media
		}
	}
	return nil
}

func (b *bot) transcribeMedia(ctx context.Context, media sizedMedia) (text string, err error) {
	if media.GetFileLength() > mediaMaxBytes {
		return "", errMediaTooLarge
	}
	data, err := b.Client.Download(ctx, media)
	if err != nil {
		return "", fmt.Errorf("baixar mídia: %w", err)
	}

	dir, err := os.MkdirTemp("", "transcrever-*")
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	in, out := filepath.Join(dir, "entrada"), filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return "", err
	}
	err = core.RunFFmpeg(ctx, "-y", "-loglevel", "error", "-i", in, "-t", strconv.Itoa(mediaMaxDuration),
		"-vn", "-ac", "1", "-ar", "16000", "-c:a", "libmp3lame", "-b:a", "32k", out)
	if err != nil {
		return "", err
	}
	audio, err := os.ReadFile(out)
	if err != nil {
		return "", err
	}
	return b.transcriber.transcribe(ctx, audio)
}

func (t *transcriber) transcribe(ctx context.Context, audio []byte) (text string, err error) {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "audio.mp3")
	if err != nil {
		return "", err
	}
	if _, err := file.Write(audio); err != nil {
		return "", err
	}
	fields := map[string]string{"model": t.model, "language": transcriptionLanguage, "response_format": "json"}
	for name, value := range fields {
		if err := form.WriteField(name, value); err != nil {
			return "", err
		}
	}
	if err := form.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	if t.key != "" {
		req.Header.Set("Authorization", "Bearer "+t.key)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()

	if resp.StatusCode != http.StatusOK {
		detail, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", errors.Join(fmt.Errorf("%s: %s", resp.Status, bytes.TrimSpace(detail)), readErr)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ler resposta: %w", err)
	}
	return strings.TrimSpace(out.Text), nil
}
