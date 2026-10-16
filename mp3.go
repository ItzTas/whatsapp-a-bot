package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const mp3Quality = "4"

type audioSource func(ctx context.Context, dir string) (file string, err error)

type sizedMedia interface {
	whatsmeow.DownloadableMessage
	GetFileLength() uint64
}

func (b *bot) mp3(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	source, found := b.findAudioSource(evt.Message)
	if !found {
		return b.sendText(ctx, chat, "Manda o link junto, tipo !mp3 https://... Ou responde um vídeo, um áudio ou uma mensagem com link com !mp3")
	}
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error { return b.extractAndSendAudio(ctx, evt, source) })
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("tirar mp3: %v", err)
	return b.sendText(ctx, chat, fmt.Sprintf("Não consegui tirar o áudio 😕 Talvez o post seja privado, o site não funcione com o bot ou o arquivo passe de %d MB ou de %d minutos", mediaMaxMB, mediaMaxDuration/60))
}

func (b *bot) findAudioSource(msg *waE2E.Message) (audioSource, bool) {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	for _, m := range []*waE2E.Message{msg, quoted} {
		if link := linkPattern.FindString(messageText(m)); link != "" {
			return linkAudio(link), true
		}
		if media := audioMedia(m); media != nil {
			return b.mediaAudio(media), true
		}
	}
	return nil, false
}

func audioMedia(m *waE2E.Message) sizedMedia {
	if video := m.GetVideoMessage(); video != nil {
		return video
	}
	if audio := m.GetAudioMessage(); audio != nil {
		return audio
	}
	return nil
}

func linkAudio(link string) audioSource {
	return func(ctx context.Context, dir string) (string, error) {
		files, err := ytDlpAudio(ctx, dir, link)
		if len(files) == 0 {
			return "", errors.Join(err, errNoMedia)
		}
		return files[0], nil
	}
}

func (b *bot) mediaAudio(media sizedMedia) audioSource {
	return func(ctx context.Context, dir string) (string, error) {
		if media.GetFileLength() > mediaMaxBytes {
			return "", errMediaTooLarge
		}
		data, err := b.client.Download(ctx, media)
		if err != nil {
			return "", fmt.Errorf("baixar mídia: %w", err)
		}
		in, out := filepath.Join(dir, "entrada"), filepath.Join(dir, "audio.mp3")
		if err := os.WriteFile(in, data, 0o600); err != nil {
			return "", err
		}
		return out, runFFmpeg(ctx, "-y", "-loglevel", "error", "-i", in, "-vn", "-c:a", "libmp3lame", "-q:a", mp3Quality, out)
	}
}

func (b *bot) extractAndSendAudio(ctx context.Context, evt *events.Message, source audioSource) (err error) {
	dir, err := os.MkdirTemp("", "mp3-*")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	file, err := source(ctx, dir)
	if err != nil {
		return err
	}
	audio, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return b.sendAudio(ctx, evt.Info.Chat, audio, replyTo(evt))
}

func (b *bot) sendAudio(ctx context.Context, chat types.JID, audio []byte, reply *waE2E.ContextInfo) error {
	up, err := b.client.Upload(ctx, audio, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("enviar áudio: %w", err)
	}
	_, err = b.client.SendMessage(ctx, chat, &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String("audio/mpeg"),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		ContextInfo:   reply,
	}})
	return err
}
