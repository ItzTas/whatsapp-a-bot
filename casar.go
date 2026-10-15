package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func (b *bot) casar(ctx context.Context, evt *events.Message) error {
	couple, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !casar @pessoa ou !casar @pessoa1 @pessoa2")
	}

	video, err := b.uploadGIF(ctx, pickGIF(b.gifs["casar"]))
	if err != nil {
		return fmt.Errorf("enviar gif: %w", err)
	}
	if rand.IntN(2) == 0 {
		couple[0], couple[1] = couple[1], couple[0]
	}
	groom, bride := couple[0], couple[1]
	news, err := b.weddingNews(ctx, evt.Info.Chat.ToNonAD(), couple)
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	caption := fmt.Sprintf("💍 @%s e @%s acabaram de se casar! Vivam os noivos 🎉\n\n🤵 Noivo: @%s\n👰 Noiva: @%s",
		groom.User, bride.User, groom.User, bride.User) + news.caption()
	mentioned := []string{groom.String(), bride.String()}
	for _, m := range news.mentions {
		mentioned = append(mentioned, m.String())
	}
	video.Caption = proto.String(b.withNicknames(ctx, evt.Info.Chat, caption))
	video.ContextInfo = &waE2E.ContextInfo{MentionedJID: mentioned}

	if _, err := b.client.SendMessage(ctx, evt.Info.Chat, &waE2E.Message{VideoMessage: video}); err != nil {
		return err
	}
	if news.renewal {
		return nil
	}
	return b.marriages.add(ctx, evt.Info.Chat, couple, time.Now())
}

func coupleFrom(sender types.JID, mentions []types.JID) ([2]types.JID, bool) {
	if len(mentions) == 2 {
		return [2]types.JID{mentions[0], mentions[1]}, true
	}
	if len(mentions) == 1 {
		return [2]types.JID{sender.ToNonAD(), mentions[0]}, true
	}
	return [2]types.JID{}, false
}
