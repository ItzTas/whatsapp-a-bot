package core

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func (b *Bot) SendImage(ctx context.Context, chat types.JID, m Outgoing, reply *waE2E.ContextInfo) error {
	up, err := b.Client.Upload(ctx, m.Data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("enviar imagem: %w", err)
	}
	image := &waE2E.ImageMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String(m.Mimetype),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		ContextInfo:   reply,
	}
	if m.Caption != "" {
		image.Caption = proto.String(m.Caption)
	}
	_, err = b.Client.SendMessage(ctx, chat, &waE2E.Message{ImageMessage: image})
	return err
}

func IsAnimatedWebP(webp []byte) bool {
	return len(webp) > 20 && string(webp[12:16]) == "VP8X" && webp[20]&0x02 != 0
}

func (b *Bot) SendSticker(ctx context.Context, chat types.JID, sticker []byte, reply *waE2E.ContextInfo) error {
	up, err := b.Client.Upload(ctx, sticker, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("enviar figurinha: %w", err)
	}
	_, err = b.Client.SendMessage(ctx, chat, &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String("image/webp"),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		Width:         proto.Uint32(512),
		Height:        proto.Uint32(512),
		IsAnimated:    proto.Bool(IsAnimatedWebP(sticker)),
		ContextInfo:   reply,
	}})
	return err
}
