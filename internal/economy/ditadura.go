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
	tyrantSavingsTaxMax = 100
	tyrantIncomeTaxMax  = 80
	tyrantShopTaxMax    = 75
	tyrantPixMax        = 50
	tyrantBailTaxMax    = 130
	tyrantBetTaxMax     = 50
	tyrantCoupMin       = 2
	tyrantCoupEvery     = 15 * time.Minute
	revoltPerPoint      = 0.0005
	revoltMax           = 0.3
	battalionGuard      = 0.05
	battalionTime       = 20 * time.Minute
	armyBattalions      = 6
	armyMax             = battalionGuard * armyBattalions
	battalionPercent    = 8
	battalionFloor      = 1_000
	revoltColumn        = "revolta"
	armyColumn          = "exercito_ate"
)

var (
	tyrannyDecree = toggleDecree{
		names: []string{"ditadura", "ditador", "estadodeexcecao", "excecao", "ai5", "autogolpe"}, column: "ditadura", emoji: "🪖",
		effect: fmt.Sprintf("vale a ditadura: os impostos por decreto passam do teto, decreto que alivia vale inteiro pro ditador e só %d%% pro povo, não existe mais !impeachment, o !golpe precisa de só %d golpistas e pode ser tentado a cada %s, todo imposto abusivo cobrado deixa o povo mais revoltado e, se o ditador morrer, perde o cargo",
			tyrantPeopleShare, tyrantCoupMin, core.FormatWait(tyrantCoupEvery)),
		lasting: "🪖 Ditadura não se desfaz por decreto. Ela só acaba com o fim do mandato, com um !golpe, com a morte do ditador ou com o !renunciar",
		syntax:  "!decreto ditadura",
		about: fmt.Sprintf("declara ditadura até o fim do mandato, sem volta: os impostos por decreto vão bem além do teto, decreto que alivia (imposto menor, espera menor, salário maior...) vale inteiro pro ditador e pro povo só chega %d%% da mudança, e o !impeachment deixa de existir, mas o !golpe fica mais fácil (%d golpistas, a cada %s), cada imposto abusivo cobrado aumenta a revolta (que tira até %d%% da segurança do ditador) e quem matar o ditador derruba ele. Pra se proteger, tem o !decreto exercito",
			tyrantPeopleShare, tyrantCoupMin, core.FormatWait(tyrantCoupEvery), percent(revoltMax)),
	}
	tyrannyColumns = []string{revoltColumn, armyColumn}
	armySyntax     = "!decreto exercito 2"
)

type armyRaise struct {
	over       bool
	full       bool
	battalions int64
	price      int64
	cost       int64
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

func (d decrees) revolt() float64 {
	return min(float64(d[revoltColumn])*revoltPerPoint, revoltMax)
}

func (d decrees) army(now time.Time) float64 {
	left := time.Unix(d[armyColumn], 0).Sub(now)
	if left <= 0 {
		return 0
	}
	return min(float64(left)/float64(battalionTime)*battalionGuard, armyMax)
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
		army = fmt.Sprintf("🪖 Exército do ditador: +%d%% de segurança, que some em %s", percent(guard), core.FormatWait(time.Unix(d[armyColumn], 0).Sub(now)))
	}
	return []string{
		fmt.Sprintf("🔥 Revolta do povo: tira %d%% da segurança do ditador (no máximo %d%%)", percent(d.revolt()), percent(revoltMax)),
		army,
	}
}

func (p Profession) tyrannical(now time.Time) Profession {
	if !p.decrees.on(tyrannyDecree) {
		return p
	}
	p.cordon += p.decrees.army(now) - p.decrees.revolt()
	return p
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

func deposedText(person types.JID) string {
	return fmt.Sprintf("🪖 @%s era o ditador e caiu junto com o corpo: o cargo ficou vago e a ditadura acabou. Manda !votar @pessoa pra abrir uma eleição nova", person.User)
}

func armyHelp() core.Usage {
	return core.Usage{armySyntax, fmt.Sprintf("só na ditadura: contrata batalhões pro exército do ditador. Cada um custa %d%% do saldo dele (no mínimo %s), soma %d%% de segurança contra !matar, !sequestrar e !golpe e deserta em %s, até %d%% de uma vez. O dinheiro some",
		battalionPercent, Coins(battalionFloor), percent(battalionGuard), core.FormatWait(battalionTime), percent(armyMax))}
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
	guard := decrees{armyColumn: r.until.Unix()}.army(call.now)
	text := fmt.Sprintf("🪖 *Exército do ditador!* @%s pagou %s por %s. A segurança dele sobe pra +%d%% e vai caindo até os últimos desertarem, daqui a %s",
		user, Coins(r.cost), core.CountLabel(int(r.battalions), "batalhão", "batalhões"), percent(guard), core.FormatWait(r.until.Sub(call.now)))
	if r.battalions < wanted {
		text += "\n_Só deu pra contratar esses: o resto não cabia no exército ou no bolso_"
	}
	return text, nil
}

func (s *walletStore) raiseArmy(ctx context.Context, chat types.JID, president core.Account, wanted int64, now time.Time) (r armyRaise, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var until, balance int64
		err := tx.QueryRowContext(ctx,
			`SELECT p.`+armyColumn+`, c.saldo FROM presidencias p JOIN carteiras c ON c.chat = p.chat AND c.pessoa = p.pessoa
			 WHERE p.chat = ? AND p.pessoa = ? AND p.mandato_ate > ? AND p.`+tyrannyDecree.column+` != 0`,
			chat.String(), president.Key, now.Unix(),
		).Scan(&until, &balance)
		if errors.Is(err, sql.ErrNoRows) {
			r.over = true
			return nil
		}
		if err != nil {
			return err
		}
		start := now
		if hired := time.Unix(until, 0); hired.After(now) {
			start = hired
		}
		room := int64((time.Duration(armyBattalions)*battalionTime - start.Sub(now)) / battalionTime)
		r.price = max(balance*battalionPercent/100, battalionFloor)
		r.until, r.full = start, room <= 0
		if r.battalions = max(min(wanted, room, balance/r.price), 0); r.battalions == 0 {
			return nil
		}
		r.cost = r.price * r.battalions
		if err := debitBalance(ctx, tx, chat, president, r.cost); err != nil {
			return err
		}
		r.until = start.Add(time.Duration(r.battalions) * battalionTime)
		_, err = tx.ExecContext(ctx,
			"UPDATE presidencias SET "+armyColumn+" = ? WHERE chat = ? AND pessoa = ?",
			r.until.Unix(), chat.String(), president.Key,
		)
		return err
	})
	return r, err
}
