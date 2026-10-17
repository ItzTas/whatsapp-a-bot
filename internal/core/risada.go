package core

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"math/rand/v2"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

//go:embed figurinhas/risada.webp
var laughSticker []byte

//go:embed figurinhas/risada*.webp
var laughStickers embed.FS

var laughWord = regexp.MustCompile(`^(k[kjs]{2,}|h?(ah|ha){2,}h?|h?(eh|he){2,}h?|(rs){2,}r?)$`)

const (
	laughReach      = 5 * time.Minute
	laughCandidates = 10
)

type laugh struct {
	target    string
	targetAlt string
	targetMsg string
}

func RandomLaughSticker() ([]byte, error) {
	names, err := fs.Glob(laughStickers, "figurinhas/risada*.webp")
	if err != nil {
		return nil, err
	}
	return laughStickers.ReadFile(names[rand.IntN(len(names))])
}

func (b *Bot) SendLaughSticker(ctx context.Context, chat types.JID, reply *waE2E.ContextInfo) error {
	return b.SendSticker(ctx, chat, laughSticker, reply)
}

func HasLaugh(text string) bool {
	words := strings.FieldsFunc(FoldLetters(text), func(r rune) bool { return !unicode.IsLetter(r) })
	return slices.ContainsFunc(words, IsLaughWord)
}

func IsLaughWord(word string) bool {
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

func (s *MessageStore) recordLaugh(ctx context.Context, evt *events.Message, text string) error {
	if !HasLaugh(text) {
		return nil
	}
	l, ok, err := s.laughTarget(ctx, evt)
	if err != nil || !ok {
		return err
	}
	_, err = s.DB.ExecContext(ctx,
		`INSERT OR IGNORE INTO risadas (chat, id, quem_riu, alvo, alvo_alt, mensagem_alvo, enviada_em)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		evt.Info.Chat.ToNonAD().String(), evt.Info.ID, evt.Info.Sender.User, l.target, l.targetAlt, l.targetMsg, evt.Info.Timestamp.Unix(),
	)
	return err
}

func (s *MessageStore) laughTarget(ctx context.Context, evt *events.Message) (laugh, bool, error) {
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

func (s *MessageStore) previousMessage(ctx context.Context, evt *events.Message) (l laugh, found bool, err error) {
	sentAt := evt.Info.Timestamp.Unix()
	rows, err := s.DB.QueryContext(ctx,
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
		if !HasLaugh(text) && !IsCommand(text) {
			return l, true, nil
		}
	}
	return laugh{}, false, rows.Err()
}

func (s *MessageStore) LaughsByTarget(ctx context.Context, chat types.JID, since int64) (counts []SenderCount, err error) {
	rows, err := s.DB.QueryContext(ctx,
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
		var c SenderCount
		if err := rows.Scan(&c.Sender, &c.SenderAlt, &c.Messages); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	return counts, rows.Err()
}
