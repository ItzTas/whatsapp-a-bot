package utilities

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	mp3Quality         = "4"
	audioThumbnailSize = "480"
)

type extractedAudio struct {
	file      string
	title     string
	thumbnail []byte
}

type audioSource func(ctx context.Context, dir string) (extractedAudio, error)

type audioSender func(ctx context.Context, evt *events.Message, audio extractedAudio) error

type sizedMedia interface {
	whatsmeow.DownloadableMessage
	GetFileLength() uint64
}

func (b *bot) mp3(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat
	source, found := b.findAudioSource(evt.Message)
	if !found {
		return b.SendText(ctx, chat, "Manda o link junto, tipo !mp3 https://... Ou responde um vídeo, um áudio ou uma mensagem com link com !mp3")
	}
	err := b.WithMediaJob(ctx, evt, func(ctx context.Context) error {
		return b.extractAndSendAudio(ctx, evt, source, b.sendAudioDocument)
	})
	if err == nil {
		return nil
	}
	b.Client.Log.Warnf("tirar mp3: %v", err)
	return b.SendText(ctx, chat, audioExtractionFailure)
}

var audioExtractionFailure = fmt.Sprintf("Não consegui tirar o áudio 😕 Talvez o post seja privado, o site não funcione com o bot ou o arquivo passe de %d MB ou de %d minutos", mediaMaxMB, mediaMaxDuration/60)

func (b *bot) findAudioSource(msg *waE2E.Message) (audioSource, bool) {
	quoted := msg.GetExtendedTextMessage().GetContextInfo().GetQuotedMessage()
	for _, m := range []*waE2E.Message{msg, quoted} {
		if link := linkPattern.FindString(core.MessageText(m)); link != "" {
			return b.linkAudio(link), true
		}
		if media := audioMedia(m); media != nil {
			return b.mediaAudio(media, m.GetVideoMessage().GetJPEGThumbnail()), true
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

func (b *bot) linkAudio(link string) audioSource {
	return func(ctx context.Context, dir string) (extractedAudio, error) {
		files, err := ytDlpAudio(ctx, dir, link)
		mp3 := slices.IndexFunc(files, func(f string) bool { return filepath.Ext(f) == ".mp3" })
		if mp3 < 0 {
			return extractedAudio{}, errors.Join(err, errNoMedia)
		}
		audio := extractedAudio{file: files[mp3], title: downloadedTitle(files)}
		thumbnail, err := downloadedThumbnail(ctx, dir, files)
		if err != nil {
			b.Client.Log.Warnf("capa do mp3: %v", err)
		}
		audio.thumbnail = thumbnail
		return audio, nil
	}
}

func downloadedTitle(files []string) string {
	i := slices.IndexFunc(files, func(f string) bool { return strings.HasSuffix(f, ".info.json") })
	if i < 0 {
		return ""
	}
	data, err := os.ReadFile(files[i])
	if err != nil {
		return ""
	}
	var info struct {
		Title string `json:"title"`
	}
	if json.Unmarshal(data, &info) != nil {
		return ""
	}
	return info.Title
}

func downloadedThumbnail(ctx context.Context, dir string, files []string) ([]byte, error) {
	i := slices.IndexFunc(files, func(f string) bool {
		return slices.Contains([]string{".webp", ".jpg", ".jpeg", ".png"}, filepath.Ext(f))
	})
	if i < 0 {
		return nil, nil
	}
	out := filepath.Join(dir, "capa.jpg")
	scale := fmt.Sprintf("scale=%s:-2", audioThumbnailSize)
	if err := core.RunFFmpeg(ctx, "-y", "-loglevel", "error", "-i", files[i], "-frames:v", "1", "-vf", scale, out); err != nil {
		return nil, err
	}
	return os.ReadFile(out)
}

func (b *bot) mediaAudio(media sizedMedia, thumbnail []byte) audioSource {
	return func(ctx context.Context, dir string) (extractedAudio, error) {
		if media.GetFileLength() > mediaMaxBytes {
			return extractedAudio{}, errMediaTooLarge
		}
		data, err := b.Client.Download(ctx, media)
		if err != nil {
			return extractedAudio{}, fmt.Errorf("baixar mídia: %w", err)
		}
		audio, err := convertToMP3(ctx, dir, data)
		audio.thumbnail = thumbnail
		return audio, err
	}
}

func convertToMP3(ctx context.Context, dir string, data []byte) (extractedAudio, error) {
	in, out := filepath.Join(dir, "entrada"), filepath.Join(dir, "audio.mp3")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return extractedAudio{}, err
	}
	err := core.RunFFmpeg(ctx, "-y", "-loglevel", "error", "-i", in, "-vn", "-c:a", "libmp3lame", "-q:a", mp3Quality, out)
	return extractedAudio{file: out}, err
}

func (b *bot) extractAndSendAudio(ctx context.Context, evt *events.Message, source audioSource, send audioSender) (err error) {
	dir, err := os.MkdirTemp("", "mp3-*")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()

	audio, err := source(ctx, dir)
	if err != nil {
		return err
	}
	return send(ctx, evt, audio)
}

func (b *bot) sendAudioDocument(ctx context.Context, evt *events.Message, audio extractedAudio) error {
	return b.sendDocument(ctx, evt.Info.Chat, core.Outgoing{
		File:      audio.file,
		Name:      cmp.Or(audio.title, "audio") + ".mp3",
		Mimetype:  "audio/mpeg",
		Thumbnail: audio.thumbnail,
	}, core.ReplyTo(evt))
}

func (b *bot) sendAudioReply(ctx context.Context, evt *events.Message, audio extractedAudio) error {
	data, err := os.ReadFile(audio.file)
	if err != nil {
		return err
	}
	return b.sendAudio(ctx, evt.Info.Chat, data, core.ReplyTo(evt))
}

func (b *bot) sendAudio(ctx context.Context, chat types.JID, audio []byte, reply *waE2E.ContextInfo) error {
	up, err := b.Client.Upload(ctx, audio, whatsmeow.MediaAudio)
	if err != nil {
		return fmt.Errorf("enviar áudio: %w", err)
	}
	_, err = b.Client.SendMessage(ctx, chat, &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
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
