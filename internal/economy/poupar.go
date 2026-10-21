package economy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	savingsRate = 5
	savingsLock = 24 * time.Hour
	savingsMin  = 100
	savingsCap  = 20_000
)

func (b *bot) poupar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()

	p, _, err := b.Wallets.presidency(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	levy := p.levy(savingsTaxDecree, acc)
	redeemed, tax, err := b.Wallets.redeem(ctx, chat, acc, now, levy)
	if err != nil {
		return fmt.Errorf("resgatar poupança: %w", err)
	}
	text, err := b.savingsReply(ctx, evt, acc, now)
	if err != nil {
		return err
	}
	people := []types.JID{person}
	if redeemed.Amount > 0 {
		text = fmt.Sprintf("🏦 @%s resgatou a poupança: voltou %s, com %s de juros%s\n\n%s",
			person.User, Coins(redeemed.Amount+redeemed.Interest-tax), Coins(redeemed.Interest-tax), savingsTaxText(tax, levy), text)
	}
	if tax > 0 {
		people = append(people, levy.payee()...)
	}
	_, err = b.SendMentions(ctx, chat, text, people)
	return err
}

func savingsTaxText(tax int64, levy PixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf(" (já tirando %s, %d%% dos juros, de imposto %s, %s)", Coins(tax), levy.percent, levy.pocket(), levy.decreed())
}

func (b *bot) savingsReply(ctx context.Context, evt *events.Message, acc core.Account, now time.Time) (string, error) {
	chat := evt.Info.Chat.ToNonAD()
	user := acc.JID.User
	current, found, err := b.Wallets.Savings(ctx, chat, acc)
	if err != nil {
		return "", fmt.Errorf("ler poupança: %w", err)
	}
	if found {
		return lockedSavingsText(user, current, now), nil
	}

	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return "", err
	}
	rate := max(int64(savingsRate)+job.interest+job.decrees.level(interestDecree), 0)
	usage := fmt.Sprintf("🏦 Pra guardar moedas: !poupar 100 (ou metade, 10%%, tudo). Elas ficam travadas por %s e voltam com %d%% de juros, até %s por vez",
		core.FormatWait(job.savingsWait()), rate, Coins(job.savingsLimit())) + job.decrees.note(interestDecree) + job.decrees.note(savingsTaxDecree)
	arg := core.FirstPlainArg(core.MessageText(evt.Message))
	if arg == "" {
		return usage, nil
	}
	balance, _, err := b.Wallets.Balance(ctx, chat, acc)
	if err != nil {
		return "", fmt.Errorf("ler saldo: %w", err)
	}
	if balance == 0 {
		return "Você tá liso 🪹 Usa !daily e !trabalhar pra ter o que guardar", nil
	}
	amount, ok := ParseStake(arg, balance)
	if !ok {
		return usage, nil
	}
	if amount > balance {
		return "Você só tem " + Coins(balance), nil
	}
	amount = min(amount, job.savingsLimit())
	if amount < savingsMin {
		return "O mínimo pra poupar é " + Coins(savingsMin), nil
	}

	deposit := Savings{Amount: amount, Interest: amount * rate / 100, due: now.Add(job.savingsWait())}
	err = b.Wallets.deposit(ctx, chat, acc, deposit)
	if errors.Is(err, ErrNoFunds) {
		return "Seu saldo mudou no meio do depósito e não dá mais 🤨 Confere no !saldo", nil
	}
	if errors.Is(err, errSavingsActive) {
		return "Você já tem uma poupança rodando, espera ela liberar", nil
	}
	if err != nil {
		return "", fmt.Errorf("guardar na poupança: %w", err)
	}
	return fmt.Sprintf("🔒 @%s guardou %s na poupança. Daqui a %s volta com %s de juros, é só mandar !poupar pra resgatar",
		user, Coins(deposit.Amount), core.FormatWait(job.savingsWait()), Coins(deposit.Interest)) + job.decrees.note(interestDecree) + job.decrees.note(savingsTaxDecree), nil
}

func lockedSavingsText(user string, sv Savings, now time.Time) string {
	return fmt.Sprintf("🔒 @%s tem %s na poupança, rendendo %s de juros. Libera em %s",
		user, Coins(sv.Amount), Coins(sv.Interest), core.FormatWait(sv.due.Sub(now)))
}
