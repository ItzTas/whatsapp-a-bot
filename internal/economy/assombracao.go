package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	hauntID         = "assombracao"
	hauntHold       = 7 * 24 * time.Hour
	hauntTaxFactor  = 2
	hauntRansomDays = 2
	carrancaItem    = "carranca-do-velho-chico"
	carrancaBoost   = 50
)

var (
	errHomeless   = errors.New("não tem casa pra assombrar")
	errNotHaunted = errors.New("a casa não tá assombrada")
)

var hauntEffect = fmt.Sprintf("só pega em quem tem imóvel. Um espírito se muda pra casa da pessoa: a proteção e os bônus do imóvel no !daily e no !trabalhar viram o contrário e o IPTU vem %dx. Purificação não espanta: o espírito só sai no fim do prazo ou quando a pessoa paga %d dias de IPTU pra quem mandou ele (!imovel exorcizar)", hauntTaxFactor, hauntRansomDays)

type haunting struct {
	author core.Account
	left   time.Duration
	ransom int64
}

func (c charm) haunts() bool {
	return c.id == hauntID
}

func (c charm) cure() string {
	if c.haunts() {
		return fmt.Sprintf("_Pra espantar o espírito: !imovel exorcizar, pagando %d dias de IPTU pra quem mandou. Purificação não adianta_", hauntRansomDays)
	}
	return cureText()
}

func haunted(p Profession) Profession {
	e, ok := estateAt(p.estate)
	if !ok || p.warded {
		return p
	}
	p.guard -= 2 * e.guard
	p.dailyBonus -= 2 * e.dailyBonus
	p.min, p.max = p.min*(100-e.payBonus)/(100+e.payBonus), p.max*(100-e.payBonus)/(100+e.payBonus)
	return p
}

func warded(p Profession) Profession {
	p.warded = true
	e, ok := estateAt(p.estate)
	if !ok {
		return p
	}
	pay := 100 + e.payBonus*(100+carrancaBoost)/100
	p.guard += e.guard * carrancaBoost / 100
	p.dailyBonus += e.dailyBonus * carrancaBoost / 100
	p.min, p.max = p.min*pay/(100+e.payBonus), p.max*pay/(100+e.payBonus)
	return p
}

func (p Profession) hauntedHome() bool {
	return slices.Contains(p.charms, hauntID) && !p.warded
}

func (e estate) ransom() int64 {
	return e.tax * hauntRansomDays
}

func iptu(level int, haunted bool) int64 {
	e, _ := estateAt(level)
	if haunted {
		return e.tax * hauntTaxFactor
	}
	return e.tax
}

func ownsHome(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account) (bool, error) {
	var level int
	err := q.QueryRowContext(ctx, "SELECT imovel FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&level)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return level > 0, err
}

func (o charmOrder) vacantFor(ctx context.Context, tx *ledgerTx, bearer core.Account) (bool, error) {
	if !o.charm.haunts() {
		return false, nil
	}
	housed, err := ownsHome(ctx, tx, o.chat, bearer)
	return !housed, err
}

func (o charmOrder) needsHome(ctx context.Context, tx *ledgerTx) error {
	vacant, err := o.vacantFor(ctx, tx, o.target)
	if err != nil {
		return err
	}
	if vacant {
		return errHomeless
	}
	return nil
}

func vacantText(s spell, caster, target types.JID) string {
	if s.backfire || s.reflected {
		return fmt.Sprintf("a praga voltou pra @%s, que não tem casa pra assombrar, e o espírito foi embora 👻", caster.User)
	}
	return fmt.Sprintf("@%s não tem casa pra assombrar e o espírito foi embora 👻", target.User)
}

func homelessHauntText(target types.JID) string {
	return fmt.Sprintf("🪹 @%s não tem casa pra assombrar. A Casa assombrada só pega em quem tem imóvel", target.User)
}

func hauntingOf(ctx context.Context, q rowQuerier, chat types.JID, acc core.Account, now time.Time) (h haunting, err error) {
	var (
		jid   string
		until int64
		level int
	)
	err = q.QueryRowContext(ctx,
		`SELECT e.autor, COALESCE(a.jid, e.autor), e.ate, c.imovel FROM encantos e
		 JOIN carteiras c ON c.chat = e.chat AND c.pessoa = e.alvo
		 LEFT JOIN carteiras a ON a.chat = e.chat AND a.pessoa = e.autor
		 WHERE e.chat = ? AND e.alvo = ? AND e.tipo = ? AND e.ate > ?`,
		chat.String(), acc.Key, hauntID, now.Unix(),
	).Scan(&h.author.Key, &jid, &until, &level)
	if errors.Is(err, sql.ErrNoRows) {
		return h, errNotHaunted
	}
	if err != nil {
		return h, err
	}
	if h.author.JID, err = types.ParseJID(jid); err != nil {
		return h, err
	}
	home, _ := estateAt(level)
	h.ransom = home.ransom()
	h.left = time.Unix(until, 0).Sub(now)
	return h, nil
}

func (s *walletStore) haunting(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (haunting, error) {
	return hauntingOf(ctx, s.db, chat, acc, now)
}

func (s *walletStore) exorcize(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (h haunting, balance int64, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if h, err = hauntingOf(ctx, tx, chat, acc, now); err != nil {
			return err
		}
		if err := openWallet(ctx, tx, chat, h.author); err != nil {
			return err
		}
		if err := moveCoins(ctx, tx, chat, acc, h.author, h.ransom); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM encantos WHERE chat = ? AND alvo = ? AND tipo = ?", chat.String(), acc.Key, hauntID); err != nil {
			return err
		}
		return tx.QueryRowContext(ctx, "SELECT saldo FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&balance)
	})
	return h, balance, err
}

func (b *bot) exorcizeHome(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	h, balance, err := b.Wallets.exorcize(ctx, chat, b.Account(ctx, person), time.Now())
	if errors.Is(err, errNotHaunted) {
		return b.SendText(ctx, chat, "🏠 Sua casa não tá assombrada, não tem espírito nenhum pra espantar")
	}
	if errors.Is(err, ErrNoFunds) {
		text := fmt.Sprintf("👻 O espírito só sai da sua casa por %s, pagos pra @%s. Você não tem isso, confere no !saldo", Coins(h.ransom), h.author.JID.User)
		_, err := b.SendMentions(ctx, chat, text, []types.JID{h.author.JID})
		return err
	}
	if err != nil {
		return fmt.Errorf("exorcizar casa: %w", err)
	}
	text := fmt.Sprintf("🕯️ @%s pagou %s pra @%s e o espírito foi embora da casa ✨\n💰 Saldo: %s", person.User, Coins(h.ransom), h.author.JID.User, Coins(balance))
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person, h.author.JID})
	return err
}

func (b *bot) hauntedHomeText(ctx context.Context, chat, person types.JID, job Profession) (string, []types.JID, error) {
	if !slices.Contains(job.charms, hauntID) {
		return "", nil, nil
	}
	h, err := b.Wallets.haunting(ctx, chat, b.Account(ctx, person), time.Now())
	if errors.Is(err, errNotHaunted) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	if job.warded {
		text := fmt.Sprintf("\n🗿 Tem um espírito que @%s mandou rondando a casa por mais %s, mas a Carranca do Velho Chico não deixa ele entrar: enquanto ela tiver na mochila, o imóvel funciona normal e o IPTU não dobra", h.author.JID.User, core.FormatWait(h.left))
		return text, []types.JID{h.author.JID}, nil
	}
	text := fmt.Sprintf("\n🏚️ A casa tá assombrada por um espírito que @%s mandou, por mais %s. Pra espantar: !imovel exorcizar, pagando %s pra quem mandou", h.author.JID.User, core.FormatWait(h.left), Coins(h.ransom))
	return text, []types.JID{h.author.JID}, nil
}
