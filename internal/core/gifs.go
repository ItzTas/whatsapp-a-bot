package core

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"path"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

//go:embed assets
var assets embed.FS

type GIF struct {
	video     string
	thumbnail string
}

func LoadGIFs() (map[string][]GIF, error) {
	dirs, err := fs.ReadDir(assets, "assets")
	if err != nil {
		return nil, err
	}
	gifs := make(map[string][]GIF, len(dirs))
	for _, d := range dirs {
		loaded, err := loadGIFDir(path.Join("assets", d.Name()))
		if err != nil {
			return nil, err
		}
		gifs[d.Name()] = loaded
	}
	return gifs, nil
}

func loadGIFDir(dir string) ([]GIF, error) {
	paths, err := fs.Glob(assets, path.Join(dir, "*.mp4"))
	if err != nil {
		return nil, err
	}
	gifs := make([]GIF, 0, len(paths))
	for _, p := range paths {
		thumbnail := strings.TrimSuffix(p, ".mp4") + ".jpg"
		if _, err := fs.Stat(assets, thumbnail); err != nil {
			return nil, err
		}
		gifs = append(gifs, GIF{video: p, thumbnail: thumbnail})
	}
	return gifs, nil
}

func PickGIF(gifs []GIF) GIF {
	return gifs[rand.IntN(len(gifs))]
}

func (b *Bot) UploadGIF(ctx context.Context, g GIF) (*waE2E.VideoMessage, error) {
	video, err := assets.ReadFile(g.video)
	if err != nil {
		return nil, err
	}
	thumbnail, err := assets.ReadFile(g.thumbnail)
	if err != nil {
		return nil, err
	}
	up, err := b.Client.Upload(ctx, video, whatsmeow.MediaVideo)
	if err != nil {
		return nil, err
	}
	return &waE2E.VideoMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String("video/mp4"),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		GifPlayback:   proto.Bool(true),
		JPEGThumbnail: thumbnail,
	}, nil
}

func (b *Bot) SendGIF(ctx context.Context, chat types.JID, category, caption string, mentions []types.JID) error {
	video, err := b.UploadGIF(ctx, PickGIF(b.GIFs[category]))
	if err != nil {
		return fmt.Errorf("enviar gif: %w", err)
	}
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	video.Caption = proto.String(b.WithNicknames(ctx, chat, caption))
	video.ContextInfo = &waE2E.ContextInfo{MentionedJID: raw}
	_, err = b.Client.SendMessage(ctx, chat, &waE2E.Message{VideoMessage: video})
	return err
}
