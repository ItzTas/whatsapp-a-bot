package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	tyrantSavingsTaxMax   = 100
	tyrantIncomeTaxMax    = 80
	tyrantShopTaxMax      = 75
	tyrantPixMax          = 50
	tyrantBailTaxMax      = 130
	tyrantBetTaxMax       = 50
	tyrantSaleTaxMax      = 25
	tyrantCoupMin         = 2
	tyrantCoupEvery       = 15 * time.Minute
	revoltPerPoint        = 0.0005
	revoltBase            = 0.1
	revoltRise            = 0.05
	revoltRiseEvery       = 15 * time.Minute
	revoltKnee            = 0.4
	revoltSlowdown        = 0.5
	battalionGuard        = 0.05
	battalionTime         = 20 * time.Minute
	armyBattalions        = 6
	armyMax               = battalionGuard * armyBattalions
	routPace              = 2
	battalionPercent      = 8
	battalionFloor        = 1_000
	tyrantEscapeBonus     = 0.4
	tyrantShelter         = 15 * time.Minute
	tyrantCitizenImmunity = 0.4
	revoltColumn          = "revolta"
	armyColumn            = "exercito_ate"
	agitationColumn       = "agitacao"
	revoltPoint           = int64(0.01 / revoltPerPoint)
	agitationCeiling      = 25 * revoltPoint
	agitationFloor        = -15 * revoltPoint
)

var (
	tyrannyDecree = toggleDecree{
		names: []string{"ditadura", "ditador", "estadodeexcecao", "excecao", "ai5", "autogolpe"}, column: "ditadura", emoji: "🪖",
		effect: fmt.Sprintf("vale a ditadura: os impostos por decreto passam do teto, decreto que alivia vale inteiro pro ditador e só %d%% pro povo, não existe mais !impeachment, o !golpe precisa de só %d golpistas e pode ser tentado a cada %s, o ditador perde a imunidade, o povo vai se revoltando com o tempo e mais ainda a cada imposto abusivo cobrado e, se o ditador morrer, perde o cargo. Quem não é presidente foge da cadeia com até %d%% a mais de chance e escapa de até %d%% das prisões, mais quanto maior a revolta, e fica %s a mais com o escudo depois de ser roubado(a)",
			tyrantPeopleShare, tyrantCoupMin, core.FormatWait(tyrantCoupEvery), percent(tyrantEscapeBonus), percent(tyrantCitizenImmunity), core.FormatWait(tyrantShelter)),
		lasting: "🪖 Ditadura não se desfaz por decreto. Ela só acaba com o fim do mandato, com um !golpe, com a morte do ditador, com ele vendido de refém no !venderrefem ou com o !renunciar",
		syntax:  "!decreto ditadura",
		about: fmt.Sprintf("declara ditadura até o fim do mandato, sem volta: os impostos por decreto vão bem além do teto, decreto que alivia (imposto menor, espera menor, salário maior...) vale inteiro pro ditador e pro povo só chega %d%% da mudança, e o !impeachment deixa de existir, mas o !golpe fica mais fácil (%d golpistas, a cada %s), o ditador perde a imunidade contra !prender e !denunciar, a revolta começa em %d%%, sobe %d%% a cada %s e mais ainda a cada imposto abusivo cobrado e a cada protesto do povo nos eventos (a repressão do ditador nos eventos abaixa ela), sem limite (passando de %d%%, sobe só %d%% do ritmo), e tira esse tanto da segurança do ditador e quem matar o ditador derruba ele. Já o povo fica largado pela polícia: quem não é presidente foge da cadeia com até %d%% a mais de chance e escapa de até %d%% das prisões (quanto maior a revolta, mais perto disso) e fica %s a mais sem poder ser roubado(a) depois de um roubo. Pra se proteger, tem o !decreto exercito. Quem foi ditador num mandato não pode declarar ditadura de novo no mandato seguinte dele",
			tyrantPeopleShare, tyrantCoupMin, core.FormatWait(tyrantCoupEvery), percent(revoltBase), percent(revoltRise), core.FormatWait(revoltRiseEvery), percent(revoltKnee), percent(revoltSlowdown), percent(tyrantEscapeBonus), percent(tyrantCitizenImmunity), core.FormatWait(tyrantShelter)),
	}
	tyrannyColumns = []string{revoltColumn, armyColumn, agitationColumn}
	armySyntax     = "!decreto exercito 2"
)

type armyRaise struct {
	over       bool
	full       bool
	routing    bool
	battalions int64
	price      int64
	cost       int64
	guard      float64
	until      time.Time
}

func (d decrees) limits(pd percentDecree) (lo, hi int64) {
	if !d.on(tyrannyDecree) {
		return pd.lo, pd.hi
	}
	return min(pd.lo, pd.tyrantLo), max(pd.hi, pd.tyrantHi)
}

func (pd percentDecree) abuse(level int64) int64 {
	return max(level-pd.hi, pd.lo-level, 0)
}

func (d decrees) reign(now time.Time) time.Duration {
	return now.Sub(time.Unix(d[tyrannyDecree.column], 0))
}

func (d decrees) brewedRevolt(now time.Time) float64 {
	return revoltBase + float64(d.reign(now)/revoltRiseEvery)*revoltRise
}

func (d decrees) taxedRevolt() float64 {
	return float64(d[revoltColumn]) * revoltPerPoint
}

func (d decrees) stirredRevolt() float64 {
	return float64(d[agitationColumn]) * revoltPerPoint
}

func (d decrees) rawRevolt(now time.Time) float64 {
	return max(d.brewedRevolt(now)+d.taxedRevolt()+d.stirredRevolt(), 0)
}

func (d decrees) revolt(now time.Time) float64 {
	return easedRevolt(d.rawRevolt(now))
}

func easedRevolt(raw float64) float64 {
	if raw <= revoltKnee {
		return raw
	}
	return revoltKnee + (raw-revoltKnee)*revoltSlowdown
}

func revoltUnrest(revolt float64) float64 {
	return min(max(revolt/revoltKnee, 0), 1)
}

func (d decrees) revoltOn(now time.Time) float64 {
	if !d.on(tyrannyDecree) {
		return 0
	}
	return d.revolt(now)
}

func (d decrees) revoltLines(now time.Time) []string {
	raw := d.rawRevolt(now)
	revolt := easedRevolt(raw)
	unrest := revoltUnrest(revolt)
	header := fmt.Sprintf("🔥 *Revolta do povo:* %s %d%%", core.MeterBar(int(unrest*100+0.5)), percent(revolt))
	if revolt > revoltKnee {
		header += fmt.Sprintf(" (passou de %d%%: agora sobe mais devagar, mas não para)", percent(revoltKnee))
	}
	lines := []string{
		header,
		fmt.Sprintf("↳ Derruba %d%% da segurança do ditador contra !matar, !sequestrar e !golpe", percent(revolt)),
		fmt.Sprintf("↳ Quem não é presidente foge da cadeia com +%d%% de chance e a polícia larga %d%% das prisões", percent(tyrantEscapeBonus*unrest), percent(tyrantCitizenImmunity*unrest)),
	}
	if origins := d.revoltOrigins(now); len(origins) > 1 {
		origin := "↳ Vem de " + core.JoinWithAnd(origins)
		if raw > revoltKnee {
			origin += fmt.Sprintf(", e o que passa de %d%% conta só %d%%", percent(revoltKnee), percent(revoltSlowdown))
		}
		lines = append(lines, origin)
	}
	next := revoltRiseEvery - d.reign(now)%revoltRiseEvery
	return append(lines, fmt.Sprintf("↳ Sobe pra %d%% em %s, e mais a cada imposto cobrado acima do teto", percent(easedRevolt(raw+revoltRise)), core.FormatWait(next)))
}

func (d decrees) revoltOrigins(now time.Time) []string {
	taxed, stirred := d.taxedRevolt(), d.stirredRevolt()
	sources := []struct {
		on   bool
		text string
	}{
		{true, fmt.Sprintf("%d%% do tempo de ditadura", percent(d.brewedRevolt(now)))},
		{taxed > 0, fmt.Sprintf("%d%% de imposto cobrado acima do teto", percent(taxed))},
		{stirred > 0, fmt.Sprintf("%d%% dos protestos do povo", percent(stirred))},
		{stirred < 0, fmt.Sprintf("menos %d%% que a repressão abafou", percent(-stirred))},
	}
	var origins []string
	for _, s := range sources {
		if s.on {
			origins = append(origins, s.text)
		}
	}
	return origins
}

func (d decrees) routedAt() time.Time {
	steps := 0
	for revoltBase+float64(steps)*revoltRise+d.taxedRevolt()+d.stirredRevolt() <= revoltKnee {
		steps++
	}
	return time.Unix(d[tyrannyDecree.column], 0).Add(time.Duration(steps) * revoltRiseEvery)
}

func (d decrees) routing(now time.Time) bool {
	return now.After(d.routedAt())
}

func (d decrees) armyClock(now time.Time) time.Time {
	routed := now.Sub(d.routedAt())
	if routed <= 0 {
		return now
	}
	return now.Add(routed * (routPace - 1))
}

func (d decrees) armyGone() time.Time {
	until, routed := time.Unix(d[armyColumn], 0), d.routedAt()
	if !until.After(routed) {
		return until
	}
	return routed.Add(until.Sub(routed) / routPace)
}

func (d decrees) armyRoom(now time.Time) (start time.Time, room int64) {
	clock := d.armyClock(now)
	start = clock
	if hired := time.Unix(d[armyColumn], 0); hired.After(clock) {
		start = hired
	}
	return start, int64((time.Duration(armyBattalions)*battalionTime - start.Sub(clock)) / battalionTime)
}

func (d decrees) army(now time.Time) float64 {
	left := time.Unix(d[armyColumn], 0).Sub(d.armyClock(now))
	if left <= 0 {
		return 0
	}
	return min(float64(left)/float64(battalionTime)*battalionGuard, armyMax)
}

func routText() string {
	return fmt.Sprintf("🔥 Com a revolta acima de %d%%, os soldados desertam %dx mais rápido", percent(revoltKnee), routPace)
}

func (d decrees) coupMin() int {
	if d.on(tyrannyDecree) {
		return tyrantCoupMin
	}
	return coupMin
}

func (d decrees) coupEvery() time.Duration {
	if d.on(tyrannyDecree) {
		return tyrantCoupEvery
	}
	return coupEvery
}

func (d decrees) tyrannyLines(now time.Time) []string {
	if !d.on(tyrannyDecree) {
		return nil
	}
	army := "🪖 Exército do ditador: nenhum. Manda !decreto exercito pra contratar"
	if guard := d.army(now); guard > 0 {
		army = fmt.Sprintf("🪖 Exército do ditador: +%d%% de segurança, que some em %s", percent(guard), core.FormatWait(d.armyGone().Sub(now)))
	}
	lines := append(d.revoltLines(now), army)
	if !d.routing(now) {
		return lines
	}
	return append(lines, "↳ "+routText())
}

func (p Profession) tyrannical(now time.Time) Profession {
	if !p.decrees.on(tyrannyDecree) {
		return p
	}
	p.cordon += p.decrees.army(now) - p.decrees.revolt(now)
	return p
}

func (p Profession) oppressed() bool {
	return !p.president && !p.protege && p.decrees.on(tyrannyDecree)
}

func (p Profession) roused(now time.Time) Profession {
	if p.oppressed() {
		p.uprising = p.decrees.revolt(now)
	}
	return p
}

func (p Profession) unrest() float64 {
	if !p.oppressed() {
		return 0
	}
	return revoltUnrest(p.uprising)
}

func (p Profession) tyrantEscape() float64 {
	return tyrantEscapeBonus * p.unrest()
}

func (p Profession) tyrantImmunity() float64 {
	return tyrantCitizenImmunity * p.unrest()
}

func (p Profession) shelterText(accuser, suspect types.JID) string {
	if p.president {
		return immunityText(accuser, suspect)
	}
	return fmt.Sprintf("🪖 @%s mandou a viatura atrás de @%s, mas na ditadura a polícia só trabalha pro ditador e largou o caso no meio do caminho. O crime continua na ficha, então dá pra tentar de novo (com a revolta de agora, a polícia larga %d%% das prisões de quem não é presidente)", accuser.User, suspect.User, percent(p.tyrantImmunity()))
}

func tyrantEscapeText(job Profession) string {
	bonus := job.tyrantEscape()
	if bonus <= 0 {
		return ""
	}
	return fmt.Sprintf("\n🪖 Com o povo revoltado, os guardas tão ocupados demais protegendo o ditador: fugir tem %d%% a mais de chance", percent(bonus))
}

func tyrantShelterOn(ctx context.Context, tx *ledgerTx, chat types.JID, victim core.Account, now time.Time) (time.Duration, error) {
	var oppressed bool
	err := tx.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM presidencias p WHERE p.chat = ? AND p.mandato_ate > ? AND p.`+tyrannyDecree.column+` != 0 AND p.pessoa != ?
		 AND NOT EXISTS (SELECT 1 FROM ditadura_favoritos f WHERE f.chat = p.chat AND f.pessoa = ? AND f.favorito_em >= p.`+tyrannyDecree.column+`))`,
		chat.String(), now.Unix(), victim.Key, victim.Key,
	).Scan(&oppressed)
	if err != nil || !oppressed {
		return 0, err
	}
	return tyrantShelter, nil
}

func (l PixLevy) decreed() string {
	if l.excess == 0 || !l.collected() {
		return "por decreto"
	}
	return "por decreto da ditadura, e o povo ficou mais revoltado 🔥"
}

func stirRevolt(ctx context.Context, tx *ledgerTx, chat types.JID, levy PixLevy) error {
	if levy.excess == 0 || !levy.collected() {
		return nil
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE presidencias SET "+revoltColumn+" = "+revoltColumn+" + ? WHERE chat = ? AND pessoa = ?",
		levy.excess, chat.String(), levy.president.Key,
	)
	return err
}

func deposeTyrant(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, now time.Time) (bool, error) {
	res, err := tx.ExecContext(ctx,
		"UPDATE presidencias SET mandato_ate = ? WHERE chat = ? AND pessoa = ? AND mandato_ate > ? AND "+tyrannyDecree.column+" != 0",
		now.Unix(), chat.String(), acc.Key, now.Unix(),
	)
	if err != nil {
		return false, err
	}
	deposed, err := res.RowsAffected()
	if err != nil || deposed == 0 {
		return false, err
	}
	_, err = openVacancyTx(ctx, tx, chat, now)
	return true, err
}

var errTyrannyRepeat = errors.New("ditadura no mandato anterior")

func (b *bot) tyrannyDecreeRun(ctx context.Context, call decreeCall) (string, error) {
	if call.p.decrees.on(tyrannyDecree) {
		return tyrannyDecree.lasting, nil
	}
	user := call.p.president.JID.User
	declared, err := b.Wallets.declareTyranny(ctx, call.chat, call.p.president, call.now)
	if errors.Is(err, errTyrannyRepeat) {
		return fmt.Sprintf("🪖 @%s já foi ditador no mandato anterior, então nesse não rola ditadura de novo. Só no próximo mandato dele", user), nil
	}
	if err != nil {
		return "", fmt.Errorf("decreto ditadura: %w", err)
	}
	if !declared {
		return termOverText, nil
	}
	return tyrannyDecree.declaredText(user), nil
}

func (s *walletStore) declareTyranny(ctx context.Context, chat types.JID, president core.Account, now time.Time) (declared bool, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var repeat bool
		err := tx.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM ditadores d JOIN presidencias p ON p.chat = d.chat AND p.pessoa = d.pessoa
			 WHERE d.chat = ? AND d.pessoa = ? AND d.ditadura_em < p.eleito_em)`,
			chat.String(), president.Key,
		).Scan(&repeat)
		if err != nil {
			return err
		}
		if repeat {
			return errTyrannyRepeat
		}
		declared, _, err = flipToggle(ctx, tx, chat, president, tyrannyDecree, now)
		if err != nil || !declared {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO ditadores (chat, pessoa, ditadura_em) VALUES (?, ?, ?)
			 ON CONFLICT (chat, pessoa) DO UPDATE SET ditadura_em = excluded.ditadura_em`,
			chat.String(), president.Key, now.Unix(),
		)
		return err
	})
	return declared, err
}

func forgetTyrant(ctx context.Context, tx *ledgerTx, chat types.JID) error {
	_, err := tx.ExecContext(ctx,
		`DELETE FROM ditadores WHERE chat = ? AND EXISTS (SELECT 1 FROM presidencias p
		 WHERE p.chat = ditadores.chat AND p.pessoa = ditadores.pessoa AND ditadores.ditadura_em < p.eleito_em)`,
		chat.String(),
	)
	return err
}

func deposedText(person types.JID) string {
	return fmt.Sprintf("🪖 @%s era o ditador e caiu junto com o corpo: o cargo ficou vago e a ditadura acabou. Manda !votar @pessoa pra abrir uma eleição nova", person.User)
}

func armyHelp() core.Usage {
	return core.Usage{armySyntax, fmt.Sprintf("só na ditadura: contrata batalhões pro exército do ditador. Cada um custa %d%% do saldo dele (no mínimo %s), soma %d%% de segurança contra !matar, !sequestrar e !golpe e deserta em %s (%dx mais rápido com a revolta acima de %d%%), até %d%% de uma vez. O dinheiro some",
		battalionPercent, Coins(battalionFloor), percent(battalionGuard), core.FormatWait(battalionTime), routPace, percent(revoltKnee), percent(armyMax))}
}

func (b *bot) armyDecree(ctx context.Context, call decreeCall) (string, error) {
	if !call.p.decrees.on(tyrannyDecree) {
		return "🪖 Exército particular é coisa de ditador. Pra montar um, primeiro manda !decreto ditadura", nil
	}
	wanted := int64(1)
	if len(call.args) > 0 {
		n, err := strconv.ParseInt(call.args[0], 10, 64)
		if err != nil || n < 1 {
			return fmt.Sprintf("🤔 \"%s\" não é um número de batalhões. Exemplo: %s", call.args[0], armySyntax), nil
		}
		wanted = n
	}
	r, err := b.Wallets.raiseArmy(ctx, call.chat, call.p.president, wanted, call.now)
	if err != nil {
		return "", fmt.Errorf("decreto exército: %w", err)
	}
	if r.over {
		return termOverText, nil
	}
	user := call.p.president.JID.User
	if r.full {
		return fmt.Sprintf("🪖 O exército de @%s já tá no tamanho máximo (+%d%% de segurança). Espera uns soldados desertarem pra contratar mais", user, percent(armyMax)), nil
	}
	if r.battalions == 0 {
		return fmt.Sprintf("💸 Cada batalhão custa %s (%d%% do saldo, no mínimo %s), e @%s não tem isso", Coins(r.price), battalionPercent, Coins(battalionFloor), user), nil
	}
	text := fmt.Sprintf("🪖 *Exército do ditador!* @%s pagou %s por %s. A segurança dele sobe pra +%d%% e vai caindo até os últimos desertarem, daqui a %s",
		user, Coins(r.cost), core.CountLabel(int(r.battalions), "batalhão", "batalhões"), percent(r.guard), core.FormatWait(r.until.Sub(call.now)))
	if r.routing {
		text += "\n" + routText()
	}
	if r.battalions < wanted {
		text += "\n_Só deu pra contratar esses: o resto não cabia no exército ou no bolso_"
	}
	return text, nil
}

func (s *walletStore) raiseArmy(ctx context.Context, chat types.JID, president core.Account, wanted int64, now time.Time) (r armyRaise, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var until, reign, revolt, agitation, balance int64
		err := tx.QueryRowContext(ctx,
			`SELECT p.`+armyColumn+`, p.`+tyrannyDecree.column+`, p.`+revoltColumn+`, p.`+agitationColumn+`, c.saldo FROM presidencias p JOIN carteiras c ON c.chat = p.chat AND c.pessoa = p.pessoa
			 WHERE p.chat = ? AND p.pessoa = ? AND p.mandato_ate > ? AND p.`+tyrannyDecree.column+` != 0`,
			chat.String(), president.Key, now.Unix(),
		).Scan(&until, &reign, &revolt, &agitation, &balance)
		if errors.Is(err, sql.ErrNoRows) {
			r.over = true
			return nil
		}
		if err != nil {
			return err
		}
		d := decrees{armyColumn: until, tyrannyDecree.column: reign, revoltColumn: revolt, agitationColumn: agitation}
		start, room := d.armyRoom(now)
		r.price = max(balance*battalionPercent/100, battalionFloor)
		r.full, r.routing = room <= 0, d.routing(now)
		if r.battalions = max(min(wanted, room, balance/r.price), 0); r.battalions == 0 {
			return nil
		}
		r.cost = r.price * r.battalions
		if err := debitBalance(ctx, tx, chat, president, r.cost); err != nil {
			return err
		}
		d[armyColumn] = start.Add(time.Duration(r.battalions) * battalionTime).Unix()
		r.guard, r.until = d.army(now), d.armyGone()
		_, err = tx.ExecContext(ctx,
			"UPDATE presidencias SET "+armyColumn+" = ? WHERE chat = ? AND pessoa = ?",
			d[armyColumn], chat.String(), president.Key,
		)
		return err
	})
	return r, err
}
