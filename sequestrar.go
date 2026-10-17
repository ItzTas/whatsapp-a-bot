package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	kidnapChance       = 0.35
	kidnapCaughtChance = 0.35
	copVictimCaught    = 0.3
	hostageHold        = 20 * time.Minute
	hostagePriceFactor = 4
	hostageSaleWait    = 3 * time.Minute
	soldHostageHold    = 7 * time.Minute
)

var kidnapCooldown = cooldown{column: "sequestro_em", every: 45 * time.Minute}

type kidnapping struct {
	lawful   bool
	holding  bool
	taken    bool
	jailed   bool
	wait     time.Duration
	abuse    powerAbuse
	success  bool
	fine     int64
	jail     time.Duration
	jailFine int64
}

type kidnapOdds struct {
	lawful  bool
	success float64
	caught  float64
	jail    time.Duration
}

type captivity struct {
	left time.Duration
	sold bool
}

type hostage struct {
	acc  account
	left time.Duration
}

type hostageSale struct {
	chat      types.JID
	seller    types.JID
	sellerAcc account
	hostage   hostage
	closes    time.Time
}

func (b *bot) sequestrar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, "Marca quem você vai sequestrar: !sequestrar @pessoa")
	}
	kidnapper, victim := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	kidnapperAcc, victimAcc := b.account(ctx, kidnapper), b.account(ctx, victim)
	if kidnapperAcc.key == victimAcc.key {
		return b.sendText(ctx, chat, "Se sequestrar? Isso se chama se trancar no quarto 🤡")
	}

	kidnapperJob, victimJob, owed, err := b.rivalsWithDebt(ctx, chat, kidnapperAcc, victimAcc)
	if err != nil {
		return err
	}
	odds := kidnapOdds{
		lawful:  owed != nil,
		success: kidnapperJob.kidnapOdds(victimJob),
		caught:  victimJob.caughtOdds(),
		jail:    profession{jailTime: baseJail}.sentence(kidnapperJob),
	}
	if kidnapperJob.slipsAway() || odds.lawful {
		odds.caught = 0
	}
	k, err := b.wallets.kidnap(ctx, chat, kidnapperAcc, victimAcc, kidnapperJob.kidnapCooldown(), kidnapperJob.abuse(kidnapAbuse), odds, time.Now())
	if err != nil {
		return fmt.Errorf("sequestrar: %w", err)
	}
	people := []types.JID{kidnapper, victim}
	abused := k.abuse.note("Abusou do poder e sequestrou sem esperar a poeira baixar")
	if k.lawful {
		abused += debtCollectionText(owed)
	}
	if k.success {
		return b.sendGIF(ctx, chat, "sequestrar", kidnapCaption(kidnapperJob, victimJob, kidnapper, victim, k.lawful)+abused, people)
	}
	_, err = b.sendMentions(ctx, chat, kidnapText(k, victimJob, kidnapper, victim)+abused, people)
	return err
}

func (p profession) kidnapOdds(victim profession) float64 {
	return clampChance(kidnapChance + p.kidnapBonus - victim.guard - p.curfew())
}

func (p profession) caughtOdds() float64 {
	if p.jailTime > 0 {
		return kidnapCaughtChance + copVictimCaught
	}
	return kidnapCaughtChance
}

func (p profession) kidnapCooldown() cooldown {
	if p.kidnapEvery == 0 {
		return p.pace(kidnapCooldown)
	}
	return p.pace(cooldown{column: kidnapCooldown.column, every: p.kidnapEvery})
}

func (p profession) ransomRange(hostage profession) (int64, int64) {
	return p.ransomOf(hostage.min), p.ransomOf(hostage.max)
}

func (p profession) ransomOf(earnings int64) int64 {
	return earnings * hostagePriceFactor * (100 + p.ransomBonus) / 100
}

func (p profession) ransom(hostage profession) int64 {
	return p.ransomOf(hostage.min + rand.Int64N(hostage.max-hostage.min+1))
}

func kidnapCaption(kidnapperJob, victimJob profession, kidnapper, victim types.JID, lawful bool) string {
	scenes := []string{
		"🚐 @%s enfiou @%s numa van branca sem placa",
		"🥔 @%s colocou @%s num saco de batata e saiu assobiando",
		"🍬 @%s ofereceu bala pra @%s e levou junto com o pacote",
		"👰 @%s carregou @%s no colo igual noiva e ninguém estranhou",
		"📦 @%s embrulhou @%s em plástico bolha e despachou pelos Correios",
		"🛒 @%s colocou @%s no carrinho do mercado e passou no caixa rápido",
		"🎣 @%s jogou uma isca de coxinha e pescou @%s",
	}
	hideouts := []string{
		"o porão da vó", "um quartinho dos fundos com Wi-Fi ruim", "um barraco no meio do mato",
		"o porta-malas de um Uno", "uma casa de praia fora de temporada", "o depósito de uma loja de R$ 1,99",
		"um motel abandonado na beira da BR", "a casinha do cachorro",
	}
	low, high := kidnapperJob.ransomRange(victimJob)
	text := fmt.Sprintf("%s\n\n🏚️ Cativeiro: %s\n⏳ @%s fica preso(a) por %s, sem poder apostar, roubar, furtar nem sequestrar, a não ser que consiga !fugir\n💰 Como @%s %s, vale de %s a %s no mercado clandestino (!venderrefem)",
		fmt.Sprintf(pickOne(scenes), kidnapper.User, victim.User),
		pickOne(hideouts),
		victim.User, formatWait(hostageHold),
		victim.User, victimJob.title, coins(low), coins(high),
	)
	if lawful {
		return text
	}
	return text + fmt.Sprintf("\n🚔 Sequestro é crime: se a polícia !prender @%s nas próximas %s, @%s é libertado(a)",
		kidnapper.User, formatWait(min(crimeWindow, hostageHold)), victim.User)
}

func kidnapText(k kidnapping, victimJob profession, kidnapper, victim types.JID) string {
	if k.holding {
		return fmt.Sprintf("🪢 @%s já tem um refém no cativeiro. Manda !venderrefem antes de pegar outro", kidnapper.User)
	}
	if k.taken {
		return fmt.Sprintf("🪢 @%s já tá em cativeiro, alguém chegou primeiro", victim.User)
	}
	if k.jailed {
		return fmt.Sprintf("🔒 @%s tá na cadeia, nem dá pra alcançar", victim.User)
	}
	if k.wait > 0 {
		return fmt.Sprintf("🚐 @%s ainda tá lavando a van do último sequestro. Espera %s pra tentar de novo", kidnapper.User, formatWait(k.wait))
	}
	text := fmt.Sprintf("🚨 @%s tentou enfiar @%s na van, mas levou uma bolsada e saiu correndo", kidnapper.User, victim.User)
	if k.fine > 0 {
		text = fmt.Sprintf("🚨 @%s tentou enfiar @%s na van, mas a vítima gritou, juntou gente e o sequestrador pagou %s de indenização", kidnapper.User, victim.User, coins(k.fine))
	}
	if k.jail > 0 {
		return text + caughtText(k, victimJob, kidnapper, victim) + fineText(k.jailFine)
	}
	if k.lawful {
		return text
	}
	return text + fmt.Sprintf("\n🚔 Tentativa de sequestro também é crime: a polícia pode !prender @%s nas próximas %s", kidnapper.User, formatWait(crimeWindow))
}

func caughtText(k kidnapping, victimJob profession, kidnapper, victim types.JID) string {
	if victimJob.jailTime > 0 {
		return fmt.Sprintf("\n👮 Só que @%s %s e deu voz de prisão na hora! @%s pegou %s de cadeia", victim.User, victimJob.title, kidnapper.User, formatWait(k.jail))
	}
	return fmt.Sprintf("\n🚓 Uma viatura passava bem na hora e @%s foi em cana: %s de cadeia", kidnapper.User, formatWait(k.jail))
}

func (b *bot) venderrefem(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	seller := evt.Info.Sender.ToNonAD()
	sellerAcc := b.account(ctx, seller)
	now := time.Now()
	h, ok, err := b.wallets.hostageOf(ctx, chat, sellerAcc, now)
	if err != nil {
		return fmt.Errorf("ler refém: %w", err)
	}
	if !ok {
		return b.sendText(ctx, chat, "Você não tem nenhum refém pra vender 🤷 Primeiro tenta !sequestrar @pessoa")
	}
	people := []types.JID{seller, h.acc.jid}
	if h.left <= hostageSaleWait {
		text := fmt.Sprintf("⌛ O cativeiro de @%s acaba em %s, não dá tempo de achar comprador. Deixa ir ou segura até o fim", h.acc.jid.User, formatWait(h.left))
		_, err = b.sendMentions(ctx, chat, text, people)
		return err
	}

	key := chat.String() + "|" + sellerAcc.key
	sale := &hostageSale{chat: chat, seller: seller, sellerAcc: sellerAcc, hostage: h, closes: now.Add(hostageSaleWait)}
	if value, busy := b.hostageSales.LoadOrStore(key, sale); busy {
		pending := value.(*hostageSale)
		text := fmt.Sprintf("🤝 @%s já tá negociando @%s no mercado clandestino. O negócio fecha em %s", seller.User, pending.hostage.acc.jid.User, formatWait(time.Until(pending.closes)))
		_, err = b.sendMentions(ctx, chat, text, []types.JID{seller, pending.hostage.acc.jid})
		return err
	}
	b.keepHostageSale(ctx, key, sale)
	time.AfterFunc(hostageSaleWait, func() { b.closeHostageSale(ctx, key, sale) })

	text := fmt.Sprintf("🕵️ @%s levou @%s pro mercado clandestino atrás de comprador. O negócio fecha em %s\n🚔 Até lá, se a polícia !prender @%s, @%s é libertado(a), e @%s ainda pode tentar !fugir",
		seller.User, h.acc.jid.User, formatWait(hostageSaleWait), seller.User, h.acc.jid.User, h.acc.jid.User)
	_, err = b.sendMentions(ctx, chat, text, people)
	return err
}

func (b *bot) closeHostageSale(ctx context.Context, key string, sale *hostageSale) {
	defer b.hostageSales.Delete(key)
	b.dropGame(ctx, hostageSaleKind, key)
	if err := b.finishHostageSale(ctx, sale); err != nil {
		b.client.Log.Errorf("vender refém: %v", err)
	}
}

const hostageSaleKind = "venderrefem"

type hostageSaleState struct {
	Chat    types.JID     `json:"chat"`
	Seller  storedAccount `json:"seller"`
	Sender  types.JID     `json:"sender"`
	Hostage storedAccount `json:"hostage"`
	Left    time.Duration `json:"left"`
	Closes  time.Time     `json:"closes"`
}

func (b *bot) keepHostageSale(ctx context.Context, key string, sale *hostageSale) {
	state := hostageSaleState{
		Chat: sale.chat, Seller: storeAccount(sale.sellerAcc), Sender: sale.seller, Hostage: storeAccount(sale.hostage.acc),
		Left: sale.hostage.left, Closes: sale.closes,
	}
	b.keepGame(ctx, hostageSaleKind, key, state, sale.closes)
}

func (b *bot) restoreHostageSale(ctx context.Context, saved savedGame) error {
	var state hostageSaleState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler venda de refém salva: %w", err)
	}
	sale := &hostageSale{
		chat: state.Chat, seller: state.Sender, sellerAcc: state.Seller.account(),
		hostage: hostage{acc: state.Hostage.account(), left: state.Left}, closes: state.Closes,
	}
	if _, busy := b.hostageSales.LoadOrStore(saved.key, sale); busy {
		return fmt.Errorf("já tem outra venda de refém em %s", saved.key)
	}
	time.AfterFunc(time.Until(sale.closes), func() { b.closeHostageSale(ctx, saved.key, sale) })
	return nil
}

func (b *bot) finishHostageSale(ctx context.Context, sale *hostageSale) error {
	h := sale.hostage
	sellerJob, hostageJob, err := b.rivalProfessions(ctx, sale.chat, sale.sellerAcc, h.acc)
	if err != nil {
		return err
	}
	price := sellerJob.ransom(hostageJob)
	left, err := b.wallets.sellHostage(ctx, sale.chat, sale.sellerAcc, h.acc, price, time.Now())
	if err != nil {
		return err
	}
	people := []types.JID{sale.seller, h.acc.jid}
	if left <= 0 {
		text := fmt.Sprintf("🏃 O negócio melou: na hora de fechar, @%s não tava mais no cativeiro de @%s", h.acc.jid.User, sale.seller.User)
		_, err = b.sendMentions(ctx, sale.chat, text, people)
		return err
	}
	h.left = left
	return b.sendGIF(ctx, sale.chat, "venderrefem", saleCaption(sellerJob, hostageJob, sale.seller, h, price), people)
}

func saleCaption(sellerJob, hostageJob profession, seller types.JID, h hostage, price int64) string {
	buyers := []string{
		"um circo do interior", "uma fábrica de salgadinho clandestina", "um coach que tava sem plateia",
		"uma tia que precisava de alguém pra lavar a louça do almoço de domingo", "um grupo de K-pop que perdeu um integrante",
		"o dono do bar, pra lavar copo", "um sheik que achou exótico", "uma seita que adora gente nova",
		"um fazendeiro que precisava de espantalho", "um streamer, pra ser moderador sem salário",
	}
	hostageUser := h.acc.jid.User
	lines := []string{
		fmt.Sprintf("🤝 @%s vendeu @%s pra %s por %s!", seller.User, hostageUser, pickOne(buyers), coins(price)),
		fmt.Sprintf("💼 @%s %s, então valia de %s a %s", hostageUser, hostageJob.title, coins(sellerJob.ransomOf(hostageJob.min)), coins(sellerJob.ransomOf(hostageJob.max))),
	}
	if sellerJob.ransomBonus > 0 {
		lines = append(lines, fmt.Sprintf("📈 @%s %s e conhece os compradores certos: vendeu %d%% mais caro", seller.User, sellerJob.title, sellerJob.ransomBonus))
	}
	lines = append(lines, fmt.Sprintf("⏳ Quem comprou só quer @%s por %s: fica trabalhando até lá, a não ser que consiga !fugir, e agora nem a polícia resgata", hostageUser, formatWait(h.left)))
	return strings.Join(lines, "\n")
}

func (b *bot) holdIfHostage(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	c, err := b.wallets.captivity(ctx, chat, b.account(ctx, person), time.Now())
	if err != nil {
		b.client.Log.Errorf("checar se tá em cativeiro: %v", err)
		return false
	}
	if c.left == 0 {
		return false
	}
	text := fmt.Sprintf("🪢 @%s tá em cativeiro e só escapa em %s. Até lá nada de apostar, roubar, furtar nem sequestrar, a não ser que consiga !fugir, que a polícia !prender quem sequestrou ou que alguém pague o !resgate", person.User, formatWait(c.left))
	if c.sold {
		text = fmt.Sprintf("📦 @%s foi vendido(a) no mercado clandestino e tá trabalhando pra quem comprou. Só sai em %s, e até lá nada de apostar, roubar, furtar nem sequestrar, a não ser que consiga !fugir ou alguém pague o !resgate", person.User, formatWait(c.left))
	}
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		b.client.Log.Errorf("avisar que tá em cativeiro: %v", err)
	}
	return true
}

func (a arrest) rescue() (string, []types.JID) {
	if a.freed.IsEmpty() {
		return "", nil
	}
	return fmt.Sprintf("\n🔓 E de quebra libertou @%s do cativeiro!", a.freed.User), []types.JID{a.freed}
}

func (s *walletStore) kidnap(ctx context.Context, chat types.JID, kidnapper, victim account, cd cooldown, abuse abuser, odds kidnapOdds, now time.Time) (k kidnapping, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		for _, acc := range []account{kidnapper, victim} {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM sequestros WHERE chat = ? AND ate <= ?", chat.String(), now.Unix()); err != nil {
			return err
		}
		var victimJail int64
		err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM sequestros WHERE chat = ? AND sequestrador = ?),
			        EXISTS (SELECT 1 FROM sequestros WHERE chat = ? AND refem = ?),
			        (SELECT preso_ate FROM carteiras WHERE chat = ? AND pessoa = ?)`,
			chat.String(), kidnapper.key, chat.String(), victim.key, chat.String(), victim.key,
		).Scan(&k.holding, &k.taken, &victimJail)
		if err != nil {
			return err
		}
		k.jailed = time.Unix(victimJail, 0).After(now)
		if k.holding || k.taken || k.jailed {
			return nil
		}
		if k.wait, k.abuse, err = stampOrAbuse(ctx, tx, chat, kidnapper, cd, abuse, now); err != nil || k.wait > 0 {
			return err
		}
		if k.lawful = odds.lawful; !k.lawful {
			if err := markCrime(ctx, tx, chat, kidnapper, now); err != nil {
				return err
			}
		}
		if k.success = rand.Float64() < odds.success; k.success {
			_, err = tx.ExecContext(ctx,
				"INSERT INTO sequestros (chat, refem, refem_jid, sequestrador, ate) VALUES (?, ?, ?, ?, ?)",
				chat.String(), victim.key, victim.jid.String(), kidnapper.key, now.Add(hostageHold).Unix(),
			)
			return err
		}
		var balance int64
		err = tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), kidnapper.key).Scan(&balance)
		if err != nil {
			return err
		}
		if k.fine = balance * (10 + rand.Int64N(11)) / 100; k.fine > 0 {
			if err := moveCoins(ctx, tx, chat, kidnapper, victim, k.fine); err != nil {
				return err
			}
		}
		if rand.Float64() >= odds.caught {
			return nil
		}
		k.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(k.jail).Unix(), chat.String(), kidnapper.key,
		)
		if err != nil {
			return err
		}
		k.jailFine, err = lockUp(ctx, tx, chat, kidnapper, now)
		return err
	})
	return k, err
}

func (s *walletStore) captivity(ctx context.Context, chat types.JID, acc account, now time.Time) (captivity, error) {
	var until int64
	var kidnapper sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT ate, sequestrador FROM sequestros WHERE chat = ? AND refem = ? AND ate > ?",
		chat.String(), acc.key, now.Unix(),
	).Scan(&until, &kidnapper)
	if errors.Is(err, sql.ErrNoRows) {
		return captivity{}, nil
	}
	if err != nil {
		return captivity{}, err
	}
	return captivity{left: time.Unix(until, 0).Sub(now), sold: !kidnapper.Valid}, nil
}

func (s *walletStore) hostageOf(ctx context.Context, chat types.JID, kidnapper account, now time.Time) (hostage, bool, error) {
	var key, jid string
	var until int64
	err := s.db.QueryRowContext(ctx,
		"SELECT refem, refem_jid, ate FROM sequestros WHERE chat = ? AND sequestrador = ? AND ate > ?",
		chat.String(), kidnapper.key, now.Unix(),
	).Scan(&key, &jid, &until)
	if errors.Is(err, sql.ErrNoRows) {
		return hostage{}, false, nil
	}
	if err != nil {
		return hostage{}, false, err
	}
	parsed, err := types.ParseJID(jid)
	if err != nil {
		return hostage{}, false, err
	}
	return hostage{acc: account{key: key, jid: parsed}, left: time.Unix(until, 0).Sub(now)}, true, nil
}

func (s *walletStore) sellHostage(ctx context.Context, chat types.JID, seller, sold account, price int64, now time.Time) (left time.Duration, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		var until int64
		err := tx.QueryRowContext(ctx,
			"UPDATE sequestros SET sequestrador = NULL, ate = MIN(ate, ?) WHERE chat = ? AND refem = ? AND sequestrador = ? AND ate > ? RETURNING ate",
			now.Add(soldHostageHold).Unix(), chat.String(), sold.key, seller.key, now.Unix(),
		).Scan(&until)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		left = time.Unix(until, 0).Sub(now)
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", price, chat.String(), seller.key)
		return err
	})
	return left, err
}

func freeHostage(ctx context.Context, tx *sql.Tx, chat types.JID, kidnapper account, now time.Time) (types.JID, error) {
	var jid string
	err := tx.QueryRowContext(ctx,
		"DELETE FROM sequestros WHERE chat = ? AND sequestrador = ? AND ate > ? RETURNING refem_jid",
		chat.String(), kidnapper.key, now.Unix(),
	).Scan(&jid)
	if errors.Is(err, sql.ErrNoRows) {
		return types.JID{}, nil
	}
	if err != nil {
		return types.JID{}, err
	}
	return types.ParseJID(jid)
}
