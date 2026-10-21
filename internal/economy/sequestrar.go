package economy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	kidnapChance       = 0.35
	kidnapCaughtChance = 0.45
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
	scuffle  scuffle
}

type kidnapOdds struct {
	lawful  bool
	success float64
	caught  float64
	jail    time.Duration
	risk    crimeRisk
}

type captivity struct {
	left time.Duration
	sold bool
}

type hostage struct {
	acc  core.Account
	left time.Duration
}

type hostageSale struct {
	chat      types.JID
	seller    types.JID
	sellerAcc core.Account
	hostage   hostage
	closes    time.Time
}

func (b *bot) sequestrar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	aim, problem := b.targetOf(ctx, evt)
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !aim.found() {
		return b.SendText(ctx, chat, "Marca quem você vai sequestrar: !sequestrar @pessoa\n"+targetHint)
	}
	kidnapper, victim := evt.Info.Sender.ToNonAD(), aim.person
	kidnapperAcc, victimAcc := b.Account(ctx, kidnapper), b.Account(ctx, victim)
	if kidnapperAcc.Key == victimAcc.Key {
		return b.SendText(ctx, chat, "Se sequestrar? Isso se chama se trancar no quarto 🤡")
	}

	kidnapperJob, victimJob, owed, err := b.rivalsWithDebt(ctx, chat, kidnapperAcc, victimAcc)
	if err != nil {
		return err
	}
	odds := kidnapOdds{
		lawful:  owed != nil,
		success: kidnapperJob.sober(kidnapperJob.kidnapOdds(victimJob, owed)),
		caught:  kidnapperJob.flagrantOdds(victimJob.caughtOdds()),
		jail:    Profession{jailTime: baseJail}.sentence(kidnapperJob),
		risk:    kidnapRisk.by(kidnapperJob).on(victimJob),
	}
	if odds.lawful {
		odds.caught = 0
	}
	k, err := b.Wallets.kidnap(ctx, chat, kidnapperAcc, victimAcc, kidnapperJob.kidnapCooldown(), kidnapperJob.abuse(kidnapAbuse), odds, time.Now())
	if err != nil {
		return fmt.Errorf("sequestrar: %w", err)
	}
	hurtText, freed := k.scuffle.text(kidnapper, victim)
	people := append([]types.JID{kidnapper, victim}, freed...)
	abused := k.abuse.note("Abusou do poder e sequestrou sem esperar a poeira baixar")
	if k.lawful {
		abused += debtCollectionText(owed)
	}
	if k.success {
		if err := b.SendGIF(ctx, chat, "sequestrar", kidnapCaption(kidnapperJob, victimJob, kidnapper, victim, k.lawful)+abused+hurtText, people); err != nil || k.scuffle.criminal.died {
			return err
		}
		b.stirEvent(ctx, evt, eventSpot{place: "!sequestrar", won: true, victim: victim})
		return nil
	}
	_, err = b.SendMentions(ctx, chat, kidnapText(k, victimJob, kidnapper, victim)+abused+hurtText, people)
	if err != nil || k.wait > 0 || k.holding || k.taken || k.jailed || k.jail > 0 || k.scuffle.criminal.died {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!sequestrar", victim: victim})
	return nil
}

func (p Profession) kidnapOdds(victim Profession, owed *debt) float64 {
	return clampChance(kidnapChance + p.kidnapBonus + p.collectionEdge(owed) - victim.violenceGuard() - p.curfew())
}

func (p Profession) caughtOdds() float64 {
	if p.jailTime > 0 {
		return kidnapCaughtChance + copVictimCaught
	}
	return kidnapCaughtChance
}

func (p Profession) kidnapCooldown() cooldown {
	if p.kidnapEvery == 0 {
		return p.pace(kidnapCooldown)
	}
	return p.pace(cooldown{column: kidnapCooldown.column, every: p.kidnapEvery})
}

func (p Profession) ransomRange(hostage Profession) (int64, int64) {
	return p.ransomOf(hostage, hostage.min), p.ransomOf(hostage, hostage.max)
}

func (p Profession) ransomOf(hostage Profession, earnings int64) int64 {
	return earnings * hostagePriceFactor * (100 + p.ransomBonus + hostage.celebrity().ransom) / 100
}

func (p Profession) ransom(hostage Profession) int64 {
	return p.ransomOf(hostage, hostage.min+rand.Int64N(hostage.max-hostage.min+1))
}

func kidnapCaption(kidnapperJob, victimJob Profession, kidnapper, victim types.JID, lawful bool) string {
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
		fmt.Sprintf(core.PickOne(scenes), kidnapper.User, victim.User),
		core.PickOne(hideouts),
		victim.User, core.FormatWait(hostageHold),
		victim.User, victimJob.title, Coins(low), Coins(high),
	) + fameRansomText(victimJob, victim.User)
	if lawful {
		return text
	}
	return text + fmt.Sprintf("\n🚔 Sequestro é crime: se a polícia !prender @%s nas próximas %s, @%s é libertado(a)",
		kidnapper.User, core.FormatWait(min(crimeWindow, hostageHold)), victim.User)
}

func soldTyrantText(person types.JID) string {
	return fmt.Sprintf("🪖 @%s era o ditador e foi vendido(a) feito mercadoria: sem ninguém no trono, o cargo ficou vago e a ditadura acabou. Manda !votar @pessoa pra abrir uma eleição nova", person.User)
}

func kidnapText(k kidnapping, victimJob Profession, kidnapper, victim types.JID) string {
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
		return fmt.Sprintf("🚐 @%s ainda tá lavando a van do último sequestro. Espera %s pra tentar de novo", kidnapper.User, core.FormatWait(k.wait))
	}
	text := fmt.Sprintf("🚨 @%s tentou enfiar @%s na van, mas levou uma bolsada e saiu correndo", kidnapper.User, victim.User)
	if k.fine > 0 {
		text = fmt.Sprintf("🚨 @%s tentou enfiar @%s na van, mas a vítima gritou, juntou gente e o sequestrador pagou %s de indenização", kidnapper.User, victim.User, Coins(k.fine))
	}
	if k.jail > 0 {
		return text + caughtText(k, victimJob, kidnapper, victim) + fineText(k.jailFine)
	}
	if k.lawful {
		return text
	}
	return text + fmt.Sprintf("\n🚔 Tentativa de sequestro também é crime: a polícia pode !prender @%s nas próximas %s", kidnapper.User, core.FormatWait(crimeWindow))
}

func caughtText(k kidnapping, victimJob Profession, kidnapper, victim types.JID) string {
	if victimJob.jailTime > 0 {
		return fmt.Sprintf("\n👮 Só que @%s %s e deu voz de prisão na hora! @%s pegou %s de cadeia", victim.User, victimJob.title, kidnapper.User, core.FormatWait(k.jail))
	}
	return fmt.Sprintf("\n🚓 Uma viatura passava bem na hora e @%s foi em cana: %s de cadeia", kidnapper.User, core.FormatWait(k.jail))
}

func (b *bot) venderrefem(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	seller := evt.Info.Sender.ToNonAD()
	sellerAcc := b.Account(ctx, seller)
	now := time.Now()
	h, ok, err := b.Wallets.hostageOf(ctx, chat, sellerAcc, now)
	if err != nil {
		return fmt.Errorf("ler refém: %w", err)
	}
	if !ok {
		return b.SendText(ctx, chat, "Você não tem nenhum refém pra vender 🤷 Primeiro tenta !sequestrar @pessoa")
	}
	people := []types.JID{seller, h.acc.JID}
	if h.left <= hostageSaleWait {
		text := fmt.Sprintf("⌛ O cativeiro de @%s acaba em %s, não dá tempo de achar comprador. Deixa ir ou segura até o fim", h.acc.JID.User, core.FormatWait(h.left))
		_, err = b.SendMentions(ctx, chat, text, people)
		return err
	}

	key := chat.String() + "|" + sellerAcc.Key
	sale := &hostageSale{chat: chat, seller: seller, sellerAcc: sellerAcc, hostage: h, closes: now.Add(hostageSaleWait)}
	if value, busy := b.hostageSales.LoadOrStore(key, sale); busy {
		pending := value.(*hostageSale)
		text := fmt.Sprintf("🤝 @%s já tá negociando @%s no mercado clandestino. O negócio fecha em %s", seller.User, pending.hostage.acc.JID.User, core.FormatWait(time.Until(pending.closes)))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{seller, pending.hostage.acc.JID})
		return err
	}
	b.keepHostageSale(ctx, key, sale)
	time.AfterFunc(hostageSaleWait, func() { b.closeHostageSale(ctx, key, sale) })

	text := fmt.Sprintf("🕵️ @%s levou @%s pro mercado clandestino atrás de comprador. O negócio fecha em %s\n🚔 Até lá, se a polícia !prender @%s, @%s é libertado(a), e @%s ainda pode tentar !fugir",
		seller.User, h.acc.JID.User, core.FormatWait(hostageSaleWait), seller.User, h.acc.JID.User, h.acc.JID.User)
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func (b *bot) closeHostageSale(ctx context.Context, key string, sale *hostageSale) {
	defer b.hostageSales.Delete(key)
	b.DropGame(ctx, hostageSaleKind, key)
	if err := b.finishHostageSale(ctx, sale); err != nil {
		b.Client.Log.Errorf("vender refém: %v", err)
	}
}

const hostageSaleKind = "venderrefem"

type hostageSaleState struct {
	Chat    types.JID          `json:"chat"`
	Seller  core.StoredAccount `json:"seller"`
	Sender  types.JID          `json:"sender"`
	Hostage core.StoredAccount `json:"hostage"`
	Left    time.Duration      `json:"left"`
	Closes  time.Time          `json:"closes"`
}

func (b *bot) keepHostageSale(ctx context.Context, key string, sale *hostageSale) {
	state := hostageSaleState{
		Chat: sale.chat, Seller: core.StoreAccount(sale.sellerAcc), Sender: sale.seller, Hostage: core.StoreAccount(sale.hostage.acc),
		Left: sale.hostage.left, Closes: sale.closes,
	}
	b.KeepGame(ctx, hostageSaleKind, key, state, sale.closes)
}

func (b *bot) restoreHostageSale(ctx context.Context, saved core.SavedGame) error {
	var state hostageSaleState
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler venda de refém salva: %w", err)
	}
	sale := &hostageSale{
		chat: state.Chat, seller: state.Sender, sellerAcc: state.Seller.Account(),
		hostage: hostage{acc: state.Hostage.Account(), left: state.Left}, closes: state.Closes,
	}
	if _, busy := b.hostageSales.LoadOrStore(saved.Key, sale); busy {
		return fmt.Errorf("já tem outra venda de refém em %s", saved.Key)
	}
	time.AfterFunc(time.Until(sale.closes), func() { b.closeHostageSale(ctx, saved.Key, sale) })
	return nil
}

func (b *bot) finishHostageSale(ctx context.Context, sale *hostageSale) error {
	h := sale.hostage
	sellerJob, hostageJob, err := b.rivalProfessions(ctx, sale.chat, sale.sellerAcc, h.acc)
	if err != nil {
		return err
	}
	price := sellerJob.ransom(hostageJob)
	sold, err := b.Wallets.sellHostage(ctx, sale.chat, sale.sellerAcc, h.acc, price, time.Now())
	if err != nil {
		return err
	}
	people := []types.JID{sale.seller, h.acc.JID}
	if sold.left <= 0 {
		text := fmt.Sprintf("🏃 O negócio melou: na hora de fechar, @%s não tava mais no cativeiro de @%s", h.acc.JID.User, sale.seller.User)
		_, err = b.SendMentions(ctx, sale.chat, text, people)
		return err
	}
	h.left = sold.left
	caption := saleCaption(sellerJob, hostageJob, sale.seller, h, price)
	if sold.deposed {
		caption += "\n" + soldTyrantText(h.acc.JID)
	}
	return b.SendGIF(ctx, sale.chat, "venderrefem", caption, people)
}

func saleCaption(sellerJob, hostageJob Profession, seller types.JID, h hostage, price int64) string {
	buyers := []string{
		"um circo do interior", "uma fábrica de salgadinho clandestina", "um coach que tava sem plateia",
		"uma tia que precisava de alguém pra lavar a louça do almoço de domingo", "um grupo de K-pop que perdeu um integrante",
		"o dono do bar, pra lavar copo", "um sheik que achou exótico", "uma seita que adora gente nova",
		"um fazendeiro que precisava de espantalho", "um streamer, pra ser moderador sem salário",
	}
	hostageUser := h.acc.JID.User
	low, high := sellerJob.ransomRange(hostageJob)
	lines := []string{
		fmt.Sprintf("🤝 @%s vendeu @%s pra %s por %s!", seller.User, hostageUser, core.PickOne(buyers), Coins(price)),
		fmt.Sprintf("💼 @%s %s, então valia de %s a %s", hostageUser, hostageJob.title, Coins(low), Coins(high)) + fameRansomText(hostageJob, hostageUser),
	}
	if sellerJob.ransomBonus > 0 {
		lines = append(lines, fmt.Sprintf("📈 @%s %s e conhece os compradores certos: vendeu %d%% mais caro", seller.User, sellerJob.title, sellerJob.ransomBonus))
	}
	lines = append(lines, fmt.Sprintf("⏳ Quem comprou só quer @%s por %s: fica trabalhando até lá, a não ser que consiga !fugir, e agora nem a polícia resgata", hostageUser, core.FormatWait(h.left)))
	return strings.Join(lines, "\n")
}

func (b *bot) holdIfHostage(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	c, err := b.Wallets.captivity(ctx, chat, b.Account(ctx, person), time.Now())
	if err != nil {
		b.Client.Log.Errorf("checar se tá em cativeiro: %v", err)
		return false
	}
	if c.left == 0 {
		return false
	}
	text := fmt.Sprintf("🪢 @%s tá em cativeiro e só escapa em %s. Até lá nada de apostar, roubar, furtar, sequestrar nem trocar de profissão, a não ser que consiga !fugir, que a polícia !prender quem sequestrou ou que alguém pague o !resgate", person.User, core.FormatWait(c.left))
	if c.sold {
		text = fmt.Sprintf("📦 @%s foi vendido(a) no mercado clandestino e tá trabalhando pra quem comprou. Só sai em %s, e até lá nada de apostar, roubar, furtar, sequestrar nem trocar de profissão, a não ser que consiga !fugir ou alguém pague o !resgate", person.User, core.FormatWait(c.left))
	}
	if _, err := b.SendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		b.Client.Log.Errorf("avisar que tá em cativeiro: %v", err)
	}
	return true
}

func (a arrest) rescue() (string, []types.JID) {
	if a.freed.IsEmpty() {
		return "", nil
	}
	return fmt.Sprintf("\n🔓 E de quebra libertou @%s do cativeiro!", a.freed.User), []types.JID{a.freed}
}

func (s *walletStore) kidnap(ctx context.Context, chat types.JID, kidnapper, victim core.Account, cd cooldown, abuse abuser, odds kidnapOdds, now time.Time) (k kidnapping, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		for _, acc := range []core.Account{kidnapper, victim} {
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
			chat.String(), kidnapper.Key, chat.String(), victim.Key, chat.String(), victim.Key,
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
		k.success = rand.Float64() < odds.success
		if err := k.settle(ctx, tx, chat, kidnapper, victim, now); err != nil {
			return err
		}
		if k.scuffle, err = odds.risk.clash(ctx, tx, chat, kidnapper, victim, k.success, k.success, now); err != nil || k.success || k.scuffle.criminal.died {
			return err
		}
		if rand.Float64() >= odds.caught {
			return nil
		}
		k.jail = odds.jail
		_, err = tx.ExecContext(ctx,
			"UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?",
			now.Add(k.jail).Unix(), chat.String(), kidnapper.Key,
		)
		if err != nil {
			return err
		}
		k.jailFine, err = lockUp(ctx, tx, chat, kidnapper, now)
		return err
	})
	return k, err
}

func (k *kidnapping) settle(ctx context.Context, tx *ledgerTx, chat types.JID, kidnapper, victim core.Account, now time.Time) error {
	if k.success {
		_, err := tx.ExecContext(ctx,
			"INSERT INTO sequestros (chat, refem, refem_jid, sequestrador, ate) VALUES (?, ?, ?, ?, ?)",
			chat.String(), victim.Key, victim.JID.String(), kidnapper.Key, now.Add(hostageHold).Unix(),
		)
		return err
	}
	var balance int64
	err := tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), kidnapper.Key).Scan(&balance)
	if err != nil {
		return err
	}
	if k.fine = balance * (10 + rand.Int64N(11)) / 100; k.fine == 0 {
		return nil
	}
	return moveCoins(ctx, tx, chat, kidnapper, victim, k.fine)
}

func (s *walletStore) captivity(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (captivity, error) {
	var until int64
	var kidnapper sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT ate, sequestrador FROM sequestros WHERE chat = ? AND refem = ? AND ate > ?",
		chat.String(), acc.Key, now.Unix(),
	).Scan(&until, &kidnapper)
	if errors.Is(err, sql.ErrNoRows) {
		return captivity{}, nil
	}
	if err != nil {
		return captivity{}, err
	}
	return captivity{left: time.Unix(until, 0).Sub(now), sold: !kidnapper.Valid}, nil
}

func (s *walletStore) hostageOf(ctx context.Context, chat types.JID, kidnapper core.Account, now time.Time) (hostage, bool, error) {
	var key, jid string
	var until int64
	err := s.db.QueryRowContext(ctx,
		"SELECT refem, refem_jid, ate FROM sequestros WHERE chat = ? AND sequestrador = ? AND ate > ?",
		chat.String(), kidnapper.Key, now.Unix(),
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
	return hostage{acc: core.Account{Key: key, JID: parsed}, left: time.Unix(until, 0).Sub(now)}, true, nil
}

type hostageSold struct {
	left    time.Duration
	deposed bool
}

func (s *walletStore) sellHostage(ctx context.Context, chat types.JID, seller, sold core.Account, price int64, now time.Time) (h hostageSold, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var until int64
		err := tx.QueryRowContext(ctx,
			"UPDATE sequestros SET sequestrador = NULL, ate = MIN(ate, ?) WHERE chat = ? AND refem = ? AND sequestrador = ? AND ate > ? RETURNING ate",
			now.Add(soldHostageHold).Unix(), chat.String(), sold.Key, seller.Key, now.Unix(),
		).Scan(&until)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		h.left = time.Unix(until, 0).Sub(now)
		_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", price, chat.String(), seller.Key)
		if err != nil {
			return err
		}
		h.deposed, err = deposeTyrant(ctx, tx, chat, sold, now)
		return err
	})
	return h, err
}

func freeHostage(ctx context.Context, tx *ledgerTx, chat types.JID, kidnapper core.Account, now time.Time) (types.JID, error) {
	var jid string
	err := tx.QueryRowContext(ctx,
		"DELETE FROM sequestros WHERE chat = ? AND sequestrador = ? AND ate > ? RETURNING refem_jid",
		chat.String(), kidnapper.Key, now.Unix(),
	).Scan(&jid)
	if errors.Is(err, sql.ErrNoRows) {
		return types.JID{}, nil
	}
	if err != nil {
		return types.JID{}, err
	}
	return types.ParseJID(jid)
}
