package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	churchFoundPercent = 10
	churchFoundFloor   = 5_000
	churchNameMax      = 30
)

var conversionCooldown = cooldown{column: "conversao_em", every: 24 * time.Hour}

var (
	errChurchTaken = errors.New("já tem uma igreja com esse nome")
	errNoChurch    = errors.New("a igreja não existe mais")
	errSameChurch  = errors.New("já é fiel dessa igreja")
	errNotFaithful = errors.New("não é fiel de igreja nenhuma")
)

type doctrine struct {
	id     string
	emoji  string
	name   string
	tithe  int64
	flavor string
	effect func(power int64) string
	apply  func(p Profession, power int64) Profession
}

var doctrines = []doctrine{
	{id: "prosperidade", emoji: "💰", name: "Prosperidade", tithe: 12, flavor: "quem dá mais recebe mais, e o líder recebe primeiro",
		effect: func(power int64) string {
			return fmt.Sprintf("o !trabalhar paga %d%% a mais", scaled(20, power))
		},
		apply: func(p Profession, power int64) Profession {
			raise := 100 + scaled(20, power)
			p.min, p.max = p.min*raise/100, p.max*raise/100
			return p
		}},
	{id: "caridade", emoji: "🤲", name: "Caridade", tithe: 5, flavor: "ajuda o próximo, principalmente quando o próximo é você",
		effect: func(power int64) string {
			return fmt.Sprintf("o !daily rende %d%% a mais", scaled(40, power))
		},
		apply: func(p Profession, power int64) Profession {
			p.dailyBonus += scaled(40, power)
			return p
		}},
	{id: "penitencia", emoji: "⛓️", name: "Penitência", tithe: 8, flavor: "pecou, rezou, tá perdoado",
		effect: func(power int64) string {
			votes := core.CountLabel(int(scaled(1, power)), "voto", "votos")
			return fmt.Sprintf("fica %d%% menos tempo na cadeia e conta como %s a mais pro seu lado no júri do !processar", scaled(20, power), votes)
		},
		apply: func(p Profession, power int64) Profession {
			p.jailFactor -= scaled(20, power)
			p.trialEdge += int(scaled(1, power))
			return p
		}},
	{id: "misterio", emoji: "🔮", name: "Mistério", tithe: 10, flavor: "ninguém sabe direito o que acontece nas reuniões",
		effect: func(power int64) string {
			return fmt.Sprintf("quem tenta !roubar, !furtar, !sequestrar ou !matar a pessoa tem %d%% menos chance, e ela recebe %d%% de volta quando perde no !apostar e no !cassino", scaled(5, power), scaled(10, power))
		},
		apply: func(p Profession, power int64) Profession {
			p.guard += float64(scaled(5, power)) / 100
			p.cashback += scaled(10, power)
			return p
		}},
	{id: "contemplacao", emoji: "📿", name: "Contemplação", tithe: 8, flavor: "medita tanto que o tempo passa mais rápido",
		effect: func(power int64) string {
			return fmt.Sprintf("espera %d%% menos no !trabalhar e no %s, e a poupança rende %s a mais", scaled(10, power), hasteCommands, core.CountLabel(int(scaled(1, power)), "ponto", "pontos"))
		},
		apply: func(p Profession, power int64) Profession {
			p.every = p.every * time.Duration(100-scaled(10, power)) / 100
			p.haste -= scaled(10, power)
			p.interest += scaled(1, power)
			return p
		}},
}

type church struct {
	id        int64
	name      string
	doctrine  doctrine
	leader    types.JID
	leaderKey string
	coffer    int64
	members   int
	temple    int
}

type tithe struct {
	church  string
	percent int64
	paid    int64
}

type departure struct {
	church    string
	heir      types.JID
	dissolved bool
	lost      int64
}

type conversion struct {
	wait  time.Duration
	price int64
	left  departure
}

func doctrineByID(id string) (doctrine, bool) {
	i := slices.IndexFunc(doctrines, func(d doctrine) bool { return d.id == id })
	if i < 0 {
		return doctrine{}, false
	}
	return doctrines[i], true
}

func findDoctrine(query string) (doctrine, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return doctrine{}, false
	}
	i := slices.IndexFunc(doctrines, func(d doctrine) bool { return strings.HasPrefix(d.id, q) })
	if i < 0 {
		return doctrine{}, false
	}
	return doctrines[i], true
}

func (d doctrine) label() string {
	return d.emoji + " *" + d.name + "*"
}

func (d doctrine) titheText() string {
	return fmt.Sprintf("%d%% do !trabalhar e do !daily", d.tithe)
}

func (d doctrine) effectAt(level int) string {
	return d.effect(templeAt(level).power)
}

func (p Profession) devoted(church, id string, level int) Profession {
	d, ok := doctrineByID(id)
	if !ok {
		return p
	}
	p = d.apply(p, templeAt(level).power)
	p.church, p.doctrineID, p.templeLevel = church, id, level
	return p
}

func churchPerks(p Profession) []string {
	d, ok := doctrineByID(p.doctrineID)
	if !ok {
		return nil
	}
	return []string{fmt.Sprintf("⛪ É fiel da *%s* (%s), de doutrina %s (já conta nas vantagens acima), e dá %s de dízimo pro cofre dela", p.church, templeAt(p.templeLevel).label(), d.label(), d.titheText())}
}

func (t tithe) text() string {
	if t.paid == 0 {
		return ""
	}
	return fmt.Sprintf("\n⛪ %s (%d%%) foram de dízimo pro cofre da *%s*", Coins(t.paid), t.percent, t.church)
}

func (d departure) text() string {
	if d.church == "" {
		return ""
	}
	return fmt.Sprintf("\n🚪 Pra isso, largou a *%s*", d.church) + d.succession()
}

func (d departure) succession() string {
	if !d.dissolved {
		return d.heirText()
	}
	text := fmt.Sprintf("\n🏚️ A *%s* ficou sem fiel nenhum e fechou as portas", d.church)
	if d.lost > 0 {
		text += fmt.Sprintf(", e os %s do cofre sumiram", Coins(d.lost))
	}
	return text
}

func (d departure) heirText() string {
	if d.heir.IsEmpty() {
		return ""
	}
	return fmt.Sprintf("\n👑 Agora quem lidera a *%s* é @%s, que tava há mais tempo na igreja", d.church, d.heir.User)
}

func (d departure) people() []types.JID {
	if d.heir.IsEmpty() {
		return nil
	}
	return []types.JID{d.heir}
}

func churchName(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		if strings.ContainsRune("*_~`", r) {
			return -1
		}
		return r
	}, raw)
	return strings.Join(strings.Fields(cleaned), " ")
}

func parseFounding(rest string) (doctrine, string, bool) {
	fields := strings.Fields(rest)
	if len(fields) < 2 {
		return doctrine{}, "", false
	}
	d, ok := findDoctrine(fields[0])
	name := churchName(core.TextAfterFields(rest, 1))
	return d, name, ok && normalizeItemName(name) != ""
}

func matchChurch(all []church, query string) (church, bool) {
	q := normalizeItemName(query)
	if q == "" {
		return church{}, false
	}
	matchers := []func(key string) bool{
		func(key string) bool { return key == q },
		func(key string) bool { return strings.HasPrefix(key, q) },
		func(key string) bool { return strings.Contains(key, q) },
	}
	for _, match := range matchers {
		i := slices.IndexFunc(all, func(c church) bool { return match(normalizeItemName(c.name)) })
		if i >= 0 {
			return all[i], true
		}
	}
	return church{}, false
}

func foundingHint() string {
	fe := areas[areaFe]
	return fmt.Sprintf("Quem é de %s %s funda a própria com !igreja fundar <doutrina> <nome>, pagando %d%% do saldo (no mínimo %s)", fe.emoji, fe.name, churchFoundPercent, Coins(churchFoundFloor))
}

func notClergyText() string {
	fe := areas[areaFe]
	var names []string
	for _, p := range professions {
		if p.area == areaFe {
			names = append(names, p.name)
		}
	}
	return fmt.Sprintf("🙏 Só quem é de %s %s funda igreja: %s. Dá pra trocar no !profissao, ou entrar numa que já existe com !igreja entrar <nome>", fe.emoji, fe.name, core.JoinWithAnd(names))
}

func conversionWaitText(wait time.Duration) string {
	return fmt.Sprintf("🙏 Você mudou de igreja faz pouco tempo. Pra entrar em outra (ou fundar uma), espera mais %s", core.FormatWait(wait))
}

func (b *bot) igreja(ctx context.Context, evt *events.Message) error {
	text := core.MentionTagPattern.ReplaceAllString(core.MessageText(evt.Message), "")
	args := strings.Fields(core.FoldLetters(text))[1:]
	if len(args) == 0 {
		return b.myChurch(ctx, evt)
	}
	actions := map[string]func(context.Context, *events.Message, string) error{
		"fundar": b.foundChurch, "criar": b.foundChurch, "abrir": b.foundChurch,
		"entrar": b.joinChurch, "converter": b.joinChurch, "seguir": b.joinChurch,
		"sair": b.leaveChurch, "largar": b.leaveChurch, "apostatar": b.leaveChurch,
		"lista": b.listChurches, "todas": b.listChurches, "igrejas": b.listChurches,
		"doutrinas": b.doctrinesMenu, "doutrina": b.doctrinesMenu,
		"ofertar": b.offerChurch, "oferta": b.offerChurch, "doar": b.offerChurch, "depositar": b.offerChurch,
		"reformar": b.rebuildTemple, "templo": b.rebuildTemple, "construir": b.rebuildTemple,
		"milagre": b.churchMiracle, "repartir": b.churchMiracle, "distribuir": b.churchMiracle,
		"desviar": b.embezzle, "desvio": b.embezzle,
	}
	action, ok := actions[args[0]]
	if !ok {
		return b.SendText(ctx, evt.Info.Chat.ToNonAD(), "Use assim: !igreja, !igreja lista, !igreja doutrinas, !igreja fundar <doutrina> <nome>, !igreja entrar <nome>, !igreja sair, !igreja ofertar <valor>, !igreja reformar, !igreja milagre <valor> ou !igreja desviar <valor>")
	}
	return action(ctx, evt, core.TextAfterFields(text, 2))
}

func (b *bot) myChurch(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	c, found, err := b.Wallets.churchOf(ctx, chat, b.Account(ctx, evt.Info.Sender.ToNonAD()))
	if err != nil {
		return fmt.Errorf("ler igreja: %w", err)
	}
	if !found {
		return b.listChurches(ctx, evt, "")
	}
	t := templeAt(c.temple)
	lines := []string{
		"⛪ *" + c.name + "*",
		fmt.Sprintf("🧱 %s, com a doutrina a %d%% da força", t.label(), t.power),
		fmt.Sprintf("📜 Doutrina %s: %s", c.doctrine.label(), c.doctrine.effectAt(c.temple)),
		"🧾 Dízimo: " + c.doctrine.titheText(),
		"👑 Líder: @" + c.leader.User,
		"🐑 " + core.CountLabel(c.members, "fiel", "fiéis"),
		"🏦 Cofre: " + Coins(c.coffer),
		nextTempleText(c.temple),
	}
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{c.leader})
	return err
}

func (b *bot) listChurches(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	all, err := b.Wallets.churches(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler igrejas: %w", err)
	}
	if len(all) == 0 {
		return b.SendText(ctx, chat, "⛪ Ainda não tem igreja nenhuma aqui. "+foundingHint()+". As doutrinas tão no !igreja doutrinas")
	}
	lines := []string{"⛪ *Igrejas do grupo*", ""}
	leaders := make([]types.JID, 0, len(all))
	for _, c := range all {
		lines = append(lines, fmt.Sprintf("*%s*: %s, cofre de %s\n    🧱 %s · 📜 %s · 👑 @%s", c.name, core.CountLabel(c.members, "fiel", "fiéis"), Coins(c.coffer), templeAt(c.temple).label(), c.doctrine.label(), c.leader.User))
		leaders = append(leaders, c.leader)
	}
	lines = append(lines, "", "_Pra entrar numa: !igreja entrar <nome>. Todo fiel ganha o efeito da doutrina e paga o dízimo dela pro cofre. "+foundingHint()+"_")
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), leaders)
	return err
}

func (b *bot) doctrinesMenu(ctx context.Context, evt *events.Message, _ string) error {
	lines := []string{
		"📜 *Doutrinas*",
		"_Quem funda a igreja escolhe a doutrina. Todo fiel ganha o efeito dela e paga o dízimo dela, uma parte do que entra no !trabalhar e no !daily, pro cofre da igreja_",
		"",
	}
	for _, d := range doctrines {
		lines = append(lines, fmt.Sprintf("%s · dízimo de %d%%\n    _%s_\n    ✨ %s", d.label(), d.tithe, d.flavor, d.effectAt(0)))
	}
	last := temples[len(temples)-1]
	lines = append(lines, "", fmt.Sprintf("_Esse é o efeito numa %s. Cada reforma do templo (!igreja reformar) deixa a doutrina mais forte, até %d%% num %s_", templeAt(0).label(), last.power, last.label()), "_"+foundingHint()+"_")
	return b.SendText(ctx, evt.Info.Chat.ToNonAD(), strings.Join(lines, "\n"))
}

func (b *bot) foundChurch(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	d, name, ok := parseFounding(rest)
	if !ok {
		return b.SendText(ctx, chat, "Use assim: !igreja fundar <doutrina> <nome>, tipo !igreja fundar prosperidade Igreja do Pix Abençoado. As doutrinas tão no !igreja doutrinas")
	}
	if utf8.RuneCountInString(name) > churchNameMax {
		return b.SendText(ctx, chat, fmt.Sprintf("Esse nome é comprido demais 😅 Cabe no máximo %d letras", churchNameMax))
	}
	acc := b.Account(ctx, person)
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	if job.area != areaFe && !job.divine {
		return b.SendText(ctx, chat, notClergyText())
	}
	cv, err := b.Wallets.foundChurch(ctx, chat, acc, name, d, time.Now())
	if errors.Is(err, ErrNoFunds) {
		return b.SendText(ctx, chat, fmt.Sprintf("Fundar igreja custa %s e você não tem tudo isso. Confere no !saldo", Coins(cv.price)))
	}
	if errors.Is(err, errChurchTaken) {
		return b.SendText(ctx, chat, "Já tem uma igreja com esse nome aqui. Escolhe outro, ou entra nela com !igreja entrar "+name)
	}
	if err != nil {
		return fmt.Errorf("fundar igreja: %w", err)
	}
	if cv.wait > 0 {
		return b.SendText(ctx, chat, conversionWaitText(cv.wait))
	}
	text := fmt.Sprintf("⛪ @%s pagou %s e fundou a *%s*, de doutrina %s!\n_%s_\n✨ Todo fiel: %s\n🧾 Dízimo: %s, pro cofre da igreja\nPra entrar: !igreja entrar %s",
		person.User, Coins(cv.price), name, d.label(), d.flavor, d.effectAt(0), d.titheText(), name) + cv.left.text()
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, cv.left.people()...))
	return err
}

func (b *bot) joinChurch(ctx context.Context, evt *events.Message, rest string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	all, err := b.Wallets.churches(ctx, chat)
	if err != nil {
		return fmt.Errorf("ler igrejas: %w", err)
	}
	c, ok := matchChurch(all, rest)
	if !ok {
		return b.SendText(ctx, chat, "Não achei essa igreja 🤔 Use assim: !igreja entrar <nome>, e as do grupo tão no !igreja lista")
	}
	cv, err := b.Wallets.joinChurch(ctx, chat, b.Account(ctx, person), c.id, time.Now())
	if errors.Is(err, errSameChurch) {
		return b.SendText(ctx, chat, fmt.Sprintf("Você já é fiel da *%s* 😇", c.name))
	}
	if errors.Is(err, errNoChurch) {
		return b.SendText(ctx, chat, fmt.Sprintf("A *%s* acabou de fechar as portas 🏚️", c.name))
	}
	if err != nil {
		return fmt.Errorf("entrar na igreja: %w", err)
	}
	if cv.wait > 0 {
		return b.SendText(ctx, chat, conversionWaitText(cv.wait))
	}
	text := fmt.Sprintf("🐑 @%s entrou pro rebanho da *%s*!\n✨ Doutrina %s: %s\n🧾 Dízimo: %s, pro cofre da igreja",
		person.User, c.name, c.doctrine.label(), c.doctrine.effectAt(c.temple), c.doctrine.titheText()) + cv.left.text()
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, cv.left.people()...))
	return err
}

func (b *bot) leaveChurch(ctx context.Context, evt *events.Message, _ string) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	left, err := b.Wallets.leaveChurch(ctx, chat, b.Account(ctx, person))
	if errors.Is(err, errNotFaithful) {
		return b.SendText(ctx, chat, notFaithfulText)
	}
	if err != nil {
		return fmt.Errorf("sair da igreja: %w", err)
	}
	text := fmt.Sprintf("🚪 @%s largou a *%s*", person.User, left.church) + left.succession()
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, left.people()...))
	return err
}

func (s *walletStore) loadChurches(ctx context.Context, where string, args ...any) (all []church, err error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT i.id, i.nome, i.doutrina, i.cofre, i.templo, i.lider, l.jid,
		        (SELECT COUNT(*) FROM carteiras f WHERE f.chat = i.chat AND f.igreja = i.id) AS fieis
		 FROM igrejas i JOIN carteiras l ON l.chat = i.chat AND l.pessoa = i.lider
		 WHERE `+where+` ORDER BY fieis DESC, i.cofre DESC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	for rows.Next() {
		var (
			c               church
			doctrineID, jid string
		)
		if err := rows.Scan(&c.id, &c.name, &doctrineID, &c.coffer, &c.temple, &c.leaderKey, &jid, &c.members); err != nil {
			return nil, err
		}
		if c.leader, err = types.ParseJID(jid); err != nil {
			return nil, err
		}
		c.doctrine, _ = doctrineByID(doctrineID)
		all = append(all, c)
	}
	return all, rows.Err()
}

func (s *walletStore) churches(ctx context.Context, chat types.JID) ([]church, error) {
	return s.loadChurches(ctx, "i.chat = ?", chat.String())
}

func (s *walletStore) churchOf(ctx context.Context, chat types.JID, acc core.Account) (church, bool, error) {
	all, err := s.loadChurches(ctx, "i.id = (SELECT igreja FROM carteiras WHERE chat = ? AND pessoa = ?)", chat.String(), acc.Key)
	if err != nil || len(all) == 0 {
		return church{}, false, err
	}
	return all[0], true, nil
}

func (s *walletStore) foundChurch(ctx context.Context, chat types.JID, acc core.Account, name string, d doctrine, now time.Time) (cv conversion, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var err error
		if cv.wait, err = stampCooldown(ctx, tx, chat, acc, conversionCooldown, now); err != nil || cv.wait > 0 {
			return err
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
		if err != nil {
			return err
		}
		cv.price = max(balance*churchFoundPercent/100, churchFoundFloor)
		if err := debitBalance(ctx, tx, chat, acc, cv.price); err != nil {
			return err
		}
		if cv.left, err = leaveFlock(ctx, tx, chat, acc); err != nil {
			return err
		}
		var id int64
		err = tx.QueryRowContext(ctx,
			`INSERT INTO igrejas (chat, chave, nome, doutrina, lider, fundada_em) VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT (chat, chave) DO NOTHING RETURNING id`,
			chat.String(), normalizeItemName(name), name, d.id, acc.Key, now.Unix(),
		).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return errChurchTaken
		}
		if err != nil {
			return err
		}
		return enroll(ctx, tx, chat, acc, id, now)
	})
	return cv, err
}

func (s *walletStore) joinChurch(ctx context.Context, chat types.JID, acc core.Account, id int64, now time.Time) (cv conversion, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		var current int64
		err := tx.QueryRowContext(ctx, "SELECT igreja FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&current)
		if err != nil {
			return err
		}
		if current == id {
			return errSameChurch
		}
		if cv.wait, err = stampCooldown(ctx, tx, chat, acc, conversionCooldown, now); err != nil || cv.wait > 0 {
			return err
		}
		var open bool
		err = tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM igrejas WHERE id = ? AND chat = ?)", id, chat.String()).Scan(&open)
		if err != nil {
			return err
		}
		if !open {
			return errNoChurch
		}
		if cv.left, err = leaveFlock(ctx, tx, chat, acc); err != nil {
			return err
		}
		return enroll(ctx, tx, chat, acc, id, now)
	})
	return cv, err
}

func (s *walletStore) leaveChurch(ctx context.Context, chat types.JID, acc core.Account) (d departure, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var err error
		if d, err = leaveFlock(ctx, tx, chat, acc); err != nil {
			return err
		}
		if d.church == "" {
			return errNotFaithful
		}
		return nil
	})
	return d, err
}

func enroll(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, id int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET igreja = ?, igreja_desde = ? WHERE chat = ? AND pessoa = ?", id, now.Unix(), chat.String(), acc.Key)
	return err
}

func leaveFlock(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account) (departure, error) {
	f, err := flockOf(ctx, tx, chat, acc)
	if errors.Is(err, errNotFaithful) {
		return departure{}, nil
	}
	if err != nil {
		return departure{}, err
	}
	d := departure{church: f.name}
	_, err = tx.ExecContext(ctx, "UPDATE carteiras SET igreja = 0, igreja_desde = 0 WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil || f.leader != acc.Key {
		return d, err
	}
	return handOver(ctx, tx, chat, f.id, f.coffer, d)
}

func handOver(ctx context.Context, tx *sql.Tx, chat types.JID, id, coffer int64, d departure) (departure, error) {
	heir, found, err := crownSuccessor(ctx, tx, chat, id, "")
	if err != nil {
		return departure{}, err
	}
	if found {
		d.heir = heir.JID
		return d, nil
	}
	d.dissolved, d.lost = true, coffer
	_, err = tx.ExecContext(ctx, "DELETE FROM igrejas WHERE id = ?", id)
	return d, err
}

func crownSuccessor(ctx context.Context, tx *sql.Tx, chat types.JID, id int64, except string) (core.Account, bool, error) {
	var (
		heir core.Account
		jid  string
	)
	err := tx.QueryRowContext(ctx,
		"SELECT pessoa, jid FROM carteiras WHERE chat = ? AND igreja = ? AND pessoa <> ? ORDER BY igreja_desde LIMIT 1",
		chat.String(), id, except,
	).Scan(&heir.Key, &jid)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Account{}, false, nil
	}
	if err != nil {
		return core.Account{}, false, err
	}
	if heir.JID, err = types.ParseJID(jid); err != nil {
		return core.Account{}, false, err
	}
	_, err = tx.ExecContext(ctx, "UPDATE igrejas SET lider = ? WHERE id = ?", heir.Key, id)
	return heir, err == nil, err
}

func payTithe(ctx context.Context, tx *sql.Tx, chat types.JID, acc core.Account, income int64) (tithe, error) {
	if income <= 0 {
		return tithe{}, nil
	}
	var (
		t          tithe
		id         int64
		doctrineID string
	)
	err := tx.QueryRowContext(ctx,
		"SELECT i.id, i.nome, i.doutrina FROM carteiras c JOIN igrejas i ON i.id = c.igreja WHERE c.chat = ? AND c.pessoa = ?",
		chat.String(), acc.Key,
	).Scan(&id, &t.church, &doctrineID)
	if errors.Is(err, sql.ErrNoRows) {
		return tithe{}, nil
	}
	if err != nil {
		return tithe{}, err
	}
	d, _ := doctrineByID(doctrineID)
	t.percent, t.paid = d.tithe, income*d.tithe/100
	if t.paid == 0 {
		return tithe{}, nil
	}
	_, err = tx.ExecContext(ctx, "UPDATE igrejas SET cofre = cofre + ? WHERE id = ?", t.paid, id)
	return t, err
}
