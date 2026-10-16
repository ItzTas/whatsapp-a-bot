package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

//go:embed figurinhas/risada.webp
var laughSticker []byte

//go:embed figurinhas/risada*.webp
var laughStickers embed.FS

var laughWord = regexp.MustCompile(`^(k[kjs]{2,}|h?(ah|ha){2,}h?|h?(eh|he){2,}h?|(rs){2,}r?)$`)

const (
	laughReach           = 5 * time.Minute
	laughCandidates      = 10
	laughStickerCooldown = 15 * time.Second
)

type laugh struct {
	target    string
	targetAlt string
	targetMsg string
}

func (b *bot) kkk(ctx context.Context, evt *events.Message) error {
	return b.sendLaughSticker(ctx, evt.Info.Chat, quotedReply(evt.Message))
}

func (b *bot) palhaco(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.sendText(ctx, chat, "O !palhaço só funciona em grupo")
	}
	period := rankingPeriodFrom(messageText(evt.Message))
	since := max(time.Now().Add(-period.length).Unix(), windowStart())

	var (
		group  *types.GroupInfo
		counts []senderCount
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { group, err = b.client.GetGroupInfo(gctx, chat); return err })
	g.Go(func() (err error) { counts, err = b.messages.laughsByTarget(gctx, chat, since); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("montar ranking de palhaços: %w", err)
	}

	ranked := b.rankMembers(group.Participants, counts)
	if len(ranked) == 0 {
		return b.sendText(ctx, chat, fmt.Sprintf("Ninguém fez o grupo rir nos %s 😐", period.label))
	}
	top := ranked[:min(len(ranked), rankingSize)]
	people := make([]types.JID, len(top))
	for i, m := range top {
		people[i] = m.jid
	}
	lines := slices.Concat(
		[]string{fmt.Sprintf("🤡 *Palhaços do grupo* nos %s", period.label), ""},
		rankingLines(top, "risada", "risadas"),
	)
	if _, err := b.sendMentions(ctx, chat, strings.Join(lines, "\n"), people); err != nil {
		return err
	}
	return b.sendLaughSticker(ctx, chat, nil)
}

func quotedReply(msg *waE2E.Message) *waE2E.ContextInfo {
	quoted := msg.GetExtendedTextMessage().GetContextInfo()
	if quoted.GetStanzaID() == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:      quoted.StanzaID,
		Participant:   quoted.Participant,
		QuotedMessage: quoted.QuotedMessage,
	}
}

func (b *bot) answerLaugh(ctx context.Context, evt *events.Message) error {
	if !hasLaugh(messageText(evt.Message)) {
		return nil
	}
	on, err := b.modes.enabled(ctx, evt.Info.Chat.ToNonAD(), modeLaughs)
	if err != nil || !on {
		return err
	}
	if !b.laughCooldownOver(evt.Info.Chat.ToNonAD()) {
		return nil
	}
	sticker, err := randomLaughSticker()
	if err != nil {
		return err
	}
	return b.sendSticker(ctx, evt.Info.Chat, sticker, nil)
}

func (b *bot) laughCooldownOver(chat types.JID) bool {
	now := time.Now()
	last, sent := b.laughStickerSent.LoadOrStore(chat.String(), now)
	if !sent {
		return true
	}
	if now.Sub(last.(time.Time)) < laughStickerCooldown {
		return false
	}
	return b.laughStickerSent.CompareAndSwap(chat.String(), last, now)
}

func randomLaughSticker() ([]byte, error) {
	names, err := fs.Glob(laughStickers, "figurinhas/risada*.webp")
	if err != nil {
		return nil, err
	}
	return laughStickers.ReadFile(names[rand.IntN(len(names))])
}

func (b *bot) sendLaughSticker(ctx context.Context, chat types.JID, reply *waE2E.ContextInfo) error {
	return b.sendSticker(ctx, chat, laughSticker, reply)
}

func (b *bot) sendSticker(ctx context.Context, chat types.JID, sticker []byte, reply *waE2E.ContextInfo) error {
	up, err := b.client.Upload(ctx, sticker, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("enviar figurinha: %w", err)
	}
	_, err = b.client.SendMessage(ctx, chat, &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
		URL:           proto.String(up.URL),
		DirectPath:    proto.String(up.DirectPath),
		MediaKey:      up.MediaKey,
		Mimetype:      proto.String("image/webp"),
		FileEncSHA256: up.FileEncSHA256,
		FileSHA256:    up.FileSHA256,
		FileLength:    proto.Uint64(up.FileLength),
		Width:         proto.Uint32(512),
		Height:        proto.Uint32(512),
		IsAnimated:    proto.Bool(isAnimatedWebP(sticker)),
		ContextInfo:   reply,
	}})
	return err
}

func hasLaugh(text string) bool {
	words := strings.FieldsFunc(foldLetters(text), func(r rune) bool { return !unicode.IsLetter(r) })
	return slices.ContainsFunc(words, isLaughWord)
}

func isLaughWord(word string) bool {
	if laughWord.MatchString(word) {
		return true
	}
	letters := utf8.RuneCountInString(word)
	ks := strings.Count(word, "k")
	if ks >= 3 && float64(ks)/float64(letters) >= 0.3 {
		return true
	}
	return letters >= 5 && strings.Count(word, "h") >= 2 && strings.Trim(word, "ahuse") == ""
}

func (s *messageStore) recordLaugh(ctx context.Context, evt *events.Message, text string) error {
	if !hasLaugh(text) {
		return nil
	}
	l, ok, err := s.laughTarget(ctx, evt)
	if err != nil || !ok {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO risadas (chat, id, quem_riu, alvo, alvo_alt, mensagem_alvo, enviada_em)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		evt.Info.Chat.ToNonAD().String(), evt.Info.ID, evt.Info.Sender.User, l.target, l.targetAlt, l.targetMsg, evt.Info.Timestamp.Unix(),
	)
	return err
}

func (s *messageStore) laughTarget(ctx context.Context, evt *events.Message) (laugh, bool, error) {
	quoted := evt.Message.GetExtendedTextMessage().GetContextInfo()
	if quoted.GetStanzaID() == "" {
		return s.previousMessage(ctx, evt)
	}
	target, err := types.ParseJID(quoted.GetParticipant())
	if err != nil || target.User == "" {
		return laugh{}, false, nil
	}
	ownMessage := target.User == evt.Info.Sender.User || target.User == evt.Info.SenderAlt.User
	return laugh{target: target.User, targetMsg: quoted.GetStanzaID()}, !ownMessage, nil
}

func (s *messageStore) previousMessage(ctx context.Context, evt *events.Message) (l laugh, found bool, err error) {
	sentAt := evt.Info.Timestamp.Unix()
	rows, err := s.db.QueryContext(ctx,
		`SELECT remetente, remetente_alt, id, texto FROM mensagens
		 WHERE chat = ? AND id != ? AND enviada_em BETWEEN ? AND ?
		   AND remetente NOT IN (?, ?)
		 ORDER BY enviada_em DESC LIMIT ?`,
		evt.Info.Chat.ToNonAD().String(), evt.Info.ID, sentAt-int64(laughReach.Seconds()), sentAt,
		evt.Info.Sender.User, evt.Info.SenderAlt.User, laughCandidates,
	)
	if err != nil {
		return laugh{}, false, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var text string
		if err := rows.Scan(&l.target, &l.targetAlt, &l.targetMsg, &text); err != nil {
			return laugh{}, false, err
		}
		if !hasLaugh(text) && !isCommand(text) {
			return l, true, nil
		}
	}
	return laugh{}, false, rows.Err()
}

func (s *messageStore) laughsByTarget(ctx context.Context, chat types.JID, since int64) (counts []senderCount, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT alvo, alvo_alt, COUNT(DISTINCT quem_riu || ' ' || mensagem_alvo) FROM risadas
		 WHERE chat = ? AND enviada_em >= ?
		 GROUP BY alvo, alvo_alt`,
		chat.String(), since,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var c senderCount
		if err := rows.Scan(&c.sender, &c.senderAlt, &c.messages); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
