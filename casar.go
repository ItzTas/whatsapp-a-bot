package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

func (b *bot) casar(ctx context.Context, evt *events.Message) error {
	couple, ok := coupleFrom(evt.Info.Sender, mentionedJIDs(evt.Message))
	if !ok {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !casar @pessoa ou !casar @pessoa1 @pessoa2")
	}
	chat := evt.Info.Chat.ToNonAD()
	consent, err := b.chatModeOn(ctx, chat, modeConsent)
	if err != nil {
		return fmt.Errorf("checar modo consentimento: %w", err)
	}
	if consent {
		return b.propose(ctx, evt, kindMarriage)
	}
	return b.wed(ctx, chat, couple, nil)
}

func (b *bot) wed(ctx context.Context, chat types.JID, couple [2]types.JID, deal *romanceProposal) error {
	var (
		adoptions []adoption
		aliases   [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { adoptions, err = b.adoptions.inChat(gctx, chat); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler família: %w", err)
	}
	if child, parent, ok := adoptiveBond(adoptions, couple, aliases); ok {
		_, err := b.sendMentions(ctx, chat, adoptiveBondText(child, parent, "casar"), couple[:])
		return err
	}

	video, err := b.uploadGIF(ctx, pickGIF(b.gifs["casar"]))
	if err != nil {
		return fmt.Errorf("enviar gif: %w", err)
	}
	if rand.IntN(2) == 0 {
		couple[0], couple[1] = couple[1], couple[0]
	}
	groom, bride := couple[0], couple[1]
	news, err := b.weddingNews(ctx, chat, couple)
	if err != nil {
		return fmt.Errorf("ler casamentos: %w", err)
	}
	paid, ok, err := b.payProposal(ctx, deal)
	if err != nil {
		return err
	}
	if !ok {
		_, err := b.sendMentions(ctx, chat, strings.Join(paid, "\n"), couple[:])
		return err
	}
	caption := fmt.Sprintf("💍 @%s e @%s acabaram de se casar! Vivam os noivos 🎉\n\n🤵 Noivo: @%s\n👰 Noiva: @%s",
		groom.User, bride.User, groom.User, bride.User) + news.caption() + paragraph(paid)
	mentioned := []string{groom.String(), bride.String()}
	for _, m := range news.mentions {
		mentioned = append(mentioned, m.String())
	}
	video.Caption = proto.String(b.withNicknames(ctx, chat, caption))
	video.ContextInfo = &waE2E.ContextInfo{MentionedJID: mentioned}

	if _, err := b.client.SendMessage(ctx, chat, &waE2E.Message{VideoMessage: video}); err != nil {
		return err
	}
	if !news.renewal {
		return b.relationships.add(ctx, chat, kindMarriage, couple, time.Now(), deal != nil)
	}
	if deal != nil && !news.marriage.consented {
		return b.relationships.confirm(ctx, news.marriage)
	}
	return nil
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
