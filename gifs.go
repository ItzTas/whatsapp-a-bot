package main

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

type gif struct {
	video     []byte
	thumbnail []byte
}

func loadGIFs() (map[string][]gif, error) {
	dirs, err := fs.ReadDir(assets, "assets")
	if err != nil {
		return nil, err
	}
	gifs := make(map[string][]gif, len(dirs))
	for _, d := range dirs {
		loaded, err := loadGIFDir(path.Join("assets", d.Name()))
		if err != nil {
			return nil, err
		}
		gifs[d.Name()] = loaded
	}
	return gifs, nil
}

func loadGIFDir(dir string) ([]gif, error) {
	paths, err := fs.Glob(assets, path.Join(dir, "*.mp4"))
	if err != nil {
		return nil, err
	}
	gifs := make([]gif, 0, len(paths))
	for _, p := range paths {
		video, err := assets.ReadFile(p)
		if err != nil {
			return nil, err
		}
		thumbnail, err := assets.ReadFile(strings.TrimSuffix(p, ".mp4") + ".jpg")
		if err != nil {
			return nil, err
		}
		gifs = append(gifs, gif{video: video, thumbnail: thumbnail})
	}
	return gifs, nil
}

func pickGIF(gifs []gif) gif {
	return gifs[rand.IntN(len(gifs))]
}

func (b *bot) uploadGIF(ctx context.Context, g gif) (*waE2E.VideoMessage, error) {
	up, err := b.client.Upload(ctx, g.video, whatsmeow.MediaVideo)
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
		JPEGThumbnail: g.thumbnail,
	}, nil
}

func (b *bot) sendGIF(ctx context.Context, chat types.JID, category, caption string, mentions []types.JID) error {
	video, err := b.uploadGIF(ctx, pickGIF(b.gifs[category]))
	if err != nil {
		return fmt.Errorf("enviar gif: %w", err)
	}
	raw := make([]string, len(mentions))
	for i, m := range mentions {
		raw[i] = m.ToNonAD().String()
	}
	video.Caption = proto.String(b.withNicknames(ctx, chat, caption))
	video.ContextInfo = &waE2E.ContextInfo{MentionedJID: raw}
	_, err = b.client.SendMessage(ctx, chat, &waE2E.Message{VideoMessage: video})
	return err
}
