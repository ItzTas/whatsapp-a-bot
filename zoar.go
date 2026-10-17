package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	clownKind        = "palhaco"
	curseKind        = "maldicao"
	nicknameLockKind = "apelido"
	outdoorKind      = "outdoor"
	cleanseKind      = "limpar"
	outdoorMaxRunes  = 300
)

type prank struct {
	floor   int64
	percent int64
	hold    time.Duration
}

var pranks = map[string]prank{
	clownKind:        {floor: 300, percent: 2, hold: 30 * time.Minute},
	nicknameLockKind: {floor: 500, percent: 3, hold: 24 * time.Hour},
	curseKind:        {floor: 1_000, percent: 5, hold: 2 * time.Hour},
	outdoorKind:      {floor: 200, percent: 1},
	cleanseKind:      {floor: 1_000, percent: 5},
}

func (p prank) price(balance int64) int64 {
	return max(p.floor, max(balance, 0)*p.percent/100)
}

type prankOrder struct {
	chat  types.JID
	payer types.JID
	acc   account
	price int64
	now   time.Time
}

func (p profession) hexed(on bool) profession {
	if !on {
		return p
	}
	p.cursed = true
	p.min /= 2
	p.max /= 2
	return p
}

func (b *bot) zoar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	payer := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, payer)
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler saldo: %w", err)
	}
	aliases := map[string]string{
		"palhaco": clownKind, "palhacada": clownKind, "circo": clownKind,
		"apelido": nicknameLockKind, "nome": nicknameLockKind,
		"maldicao": curseKind, "praga": curseKind, "macumba": curseKind,
		"outdoor": outdoorKind, "anuncio": outdoorKind, "propaganda": outdoorKind,
		"limpar": cleanseKind, "benzer": cleanseKind, "descarrego": cleanseKind,
	}
	fields := strings.Fields(messageText(evt.Message))
	if len(fields) < 2 {
		return b.sendText(ctx, chat, prankMenu(balance))
	}
	kind, ok := aliases[foldLetters(fields[1])]
	if !ok {
		return b.sendText(ctx, chat, prankMenu(balance))
	}
	run := map[string]func(context.Context, *events.Message, prankOrder) error{
		clownKind:        b.prankClown,
		nicknameLockKind: b.prankNickname,
		curseKind:        b.prankCurse,
		outdoorKind:      b.prankOutdoor,
		cleanseKind:      b.prankCleanse,
	}
	return run[kind](ctx, evt, prankOrder{chat: chat, payer: payer, acc: acc, price: pranks[kind].price(balance), now: time.Now()})
}

func prankMenu(balance int64) string {
	price := func(kind string) string { return coins(pranks[kind].price(balance)) }
	lines := []string{
		"🃏 *Zoeira paga*",
		"_O preço sobe com o seu saldo, e as moedas somem_",
		"",
		fmt.Sprintf("🤡 !zoar palhaco @pessoa (%s): o bot reage com 🤡 em toda mensagem da pessoa por %s", price(clownKind), formatWait(pranks[clownKind].hold)),
		fmt.Sprintf("🏷️ !zoar apelido @pessoa <apelido> (%s): dá um apelido e tranca por %s, ninguém consegue trocar no !apelido", price(nicknameLockKind), formatWait(pranks[nicknameLockKind].hold)),
		fmt.Sprintf("🧿 !zoar maldicao @pessoa (%s): o !trabalhar da pessoa paga metade por %s", price(curseKind), formatWait(pranks[curseKind].hold)),
		fmt.Sprintf("📢 !zoar outdoor <texto> (%s): o bot publica um outdoor com o seu texto, até %d letras", price(outdoorKind), outdoorMaxRunes),
		fmt.Sprintf("🧂 !zoar limpar (%s): faz um descarrego e tira toda zoeira que tá em você", price(cleanseKind)),
	}
	return strings.Join(lines, "\n")
}

func (b *bot) prankTarget(ctx context.Context, evt *events.Message, o prankOrder, usage string) (types.JID, bool, error) {
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return types.JID{}, false, b.sendText(ctx, o.chat, "Marca quem vai ser zoado: "+usage)
	}
	target := mentions[0].ToNonAD()
	return target, true, nil
}

func (b *bot) placePrank(ctx context.Context, o prankOrder, target types.JID, kind string) (until time.Time, refused bool, err error) {
	until, err = b.wallets.prank(ctx, o, b.account(ctx, target), kind, pranks[kind].hold)
	if errors.Is(err, errNoFunds) {
		return until, true, b.sendText(ctx, o.chat, fmt.Sprintf("Essa zoeira custa %s e você não tem tudo isso. Confere no !saldo", coins(o.price)))
	}
	if err != nil {
		return until, true, fmt.Errorf("pagar zoeira: %w", err)
	}
	return until, false, nil
}

func (b *bot) prankClown(ctx context.Context, evt *events.Message, o prankOrder) error {
	target, ok, err := b.prankTarget(ctx, evt, o, "!zoar palhaco @pessoa")
	if !ok {
		return err
	}
	until, refused, err := b.placePrank(ctx, o, target, clownKind)
	if refused {
		return err
	}
	text := fmt.Sprintf("🤡 @%s pagou %s e contratou um circo inteiro pra seguir @%s. Toda mensagem vai ganhar um 🤡 pelos próximos %s", o.payer.User, coins(o.price), target.User, formatWait(until.Sub(o.now)))
	_, err = b.sendMentions(ctx, o.chat, text, []types.JID{o.payer, target})
	return err
}

func (b *bot) prankCurse(ctx context.Context, evt *events.Message, o prankOrder) error {
	target, ok, err := b.prankTarget(ctx, evt, o, "!zoar maldicao @pessoa")
	if !ok {
		return err
	}
	until, refused, err := b.placePrank(ctx, o, target, curseKind)
	if refused {
		return err
	}
	text := fmt.Sprintf("🧿 @%s pagou %s pra uma mãe de santo e jogou uma praga em @%s! Pelos próximos %s, o !trabalhar paga só metade. Pra se livrar: !zoar limpar", o.payer.User, coins(o.price), target.User, formatWait(until.Sub(o.now)))
	_, err = b.sendMentions(ctx, o.chat, text, []types.JID{o.payer, target})
	return err
}

func (b *bot) prankNickname(ctx context.Context, evt *events.Message, o prankOrder) error {
	target, ok, err := b.prankTarget(ctx, evt, o, "!zoar apelido @pessoa <apelido>")
	if !ok {
		return err
	}
	fields := strings.Fields(mentionTagPattern.ReplaceAllString(messageText(evt.Message), ""))
	nickname := strings.Join(fields[min(2, len(fields)):], " ")
	if nickname == "" {
		return b.sendText(ctx, o.chat, "Faltou o apelido: !zoar apelido @pessoa <apelido>")
	}
	if utf8.RuneCountInString(nickname) > maxNicknameLength {
		return b.sendText(ctx, o.chat, fmt.Sprintf("Apelido grande demais, o máximo é %d letras", maxNicknameLength))
	}
	until, refused, err := b.placePrank(ctx, o, target, nicknameLockKind)
	if refused {
		return err
	}
	if err := b.nicknames.set(ctx, o.chat, b.userAliases(ctx, target), nickname); err != nil {
		return fmt.Errorf("salvar apelido pago: %w", err)
	}
	text := fmt.Sprintf("🏷️ @%s pagou %s e agora @%s se chama *%s* aqui. Tá trancado por %s e ninguém consegue trocar no !apelido, nem a vítima (só com !zoar limpar)", o.payer.User, coins(o.price), target.User, nickname, formatWait(until.Sub(o.now)))
	_, err = b.sendMentions(ctx, o.chat, text, []types.JID{o.payer, target})
	return err
}

func (b *bot) prankOutdoor(ctx context.Context, evt *events.Message, o prankOrder) error {
	fields := strings.Fields(messageText(evt.Message))
	ad := strings.Join(fields[min(2, len(fields)):], " ")
	if ad == "" {
		return b.sendText(ctx, o.chat, "Faltou o texto do outdoor: !zoar outdoor <texto>")
	}
	if utf8.RuneCountInString(ad) > outdoorMaxRunes {
		return b.sendText(ctx, o.chat, fmt.Sprintf("Outdoor grande demais, o máximo é %d letras", outdoorMaxRunes))
	}
	err := b.wallets.spend(ctx, o.chat, o.acc, o.price, func(*sql.Tx) error { return nil })
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, o.chat, fmt.Sprintf("O outdoor custa %s e você não tem tudo isso. Confere no !saldo", coins(o.price)))
	}
	if err != nil {
		return fmt.Errorf("pagar outdoor: %w", err)
	}
	text := fmt.Sprintf("📢 *OUTDOOR* 📢\n\n%s\n\n_Espaço pago por @%s (%s). Anuncie você também: !zoar outdoor_", ad, o.payer.User, coins(o.price))
	_, err = b.sendMentions(ctx, o.chat, text, append(mentionedJIDs(evt.Message), o.payer))
	return err
}

func (b *bot) prankCleanse(ctx context.Context, _ *events.Message, o prankOrder) error {
	err := b.wallets.spend(ctx, o.chat, o.acc, o.price, func(tx *sql.Tx) error {
		return releaseHeld(ctx, tx, "DELETE FROM zoeiras WHERE chat = ? AND alvo = ? AND ate > ?", o.chat, o.acc, o.now)
	})
	if errors.Is(err, errNotHeld) {
		_, err := b.sendMentions(ctx, o.chat, fmt.Sprintf("😌 @%s tá limpo(a), não tem zoeira nenhuma pra tirar", o.payer.User), []types.JID{o.payer})
		return err
	}
	if errors.Is(err, errNoFunds) {
		return b.sendText(ctx, o.chat, fmt.Sprintf("O descarrego custa %s e você não tem tudo isso. Confere no !saldo", coins(o.price)))
	}
	if err != nil {
		return fmt.Errorf("pagar descarrego: %w", err)
	}
	text := fmt.Sprintf("🧂 @%s pagou %s num descarrego com sal grosso e arruda. Toda zoeira saiu do corpo!", o.payer.User, coins(o.price))
	_, err = b.sendMentions(ctx, o.chat, text, []types.JID{o.payer})
	return err
}

func (b *bot) nicknameLocked(ctx context.Context, chat, sender, target types.JID) (string, error) {
	now := time.Now()
	author, found, err := b.wallets.prankOn(ctx, chat.ToNonAD(), b.account(ctx, target), nicknameLockKind, now)
	if err != nil || !found {
		return "", err
	}
	authorJID, err := types.ParseJID(author.jid)
	if err != nil {
		return "", err
	}
	if b.account(ctx, authorJID).key == b.account(ctx, sender).key {
		return "", nil
	}
	return fmt.Sprintf("🔒 O apelido de @%s foi trancado no !zoar e só libera em %s. Quem tá zoado pode pagar um !zoar limpar", target.User, formatWait(author.left)), nil
}

type prankAuthor struct {
	jid  string
	left time.Duration
}

func (s *walletStore) prank(ctx context.Context, o prankOrder, target account, kind string, hold time.Duration) (until time.Time, err error) {
	err = s.spend(ctx, o.chat, o.acc, o.price, func(tx *sql.Tx) error {
		var end int64
		err := tx.QueryRowContext(ctx,
			`INSERT INTO zoeiras (chat, alvo, tipo, autor, ate) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, alvo, tipo) DO UPDATE SET autor = excluded.autor, ate = MAX(zoeiras.ate, ?) + ?
			 RETURNING ate`,
			o.chat.String(), target.key, kind, o.payer.String(), o.now.Add(hold).Unix(), o.now.Unix(), int64(hold.Seconds()),
		).Scan(&end)
		until = time.Unix(end, 0)
		return err
	})
	return until, err
}

func (s *walletStore) prankOn(ctx context.Context, chat types.JID, target account, kind string, now time.Time) (prankAuthor, bool, error) {
	var (
		author string
		until  int64
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT autor, ate FROM zoeiras WHERE chat = ? AND alvo = ? AND tipo = ? AND ate > ?",
		chat.String(), target.key, kind, now.Unix(),
	).Scan(&author, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return prankAuthor{}, false, nil
	}
	if err != nil {
		return prankAuthor{}, false, err
	}
	return prankAuthor{jid: author, left: time.Unix(until, 0).Sub(now)}, true, nil
}
