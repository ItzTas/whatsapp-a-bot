package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	savingsRate = 5
	savingsLock = 24 * time.Hour
	savingsMin  = 100
	savingsCap  = 5000
)

func (b *bot) poupar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.account(ctx, person)
	now := time.Now()

	p, elected, err := b.wallets.presidency(ctx, chat, now)
	if err != nil {
		return fmt.Errorf("ler presidência: %w", err)
	}
	var levy pixLevy
	if elected && acc.key != p.president.key {
		levy = pixLevy{president: p.president, percent: p.decrees.level(savingsTaxDecree)}
	}
	redeemed, tax, err := b.wallets.redeem(ctx, chat, acc, now, levy)
	if err != nil {
		return fmt.Errorf("resgatar poupança: %w", err)
	}
	text, err := b.savingsReply(ctx, evt, acc, now)
	if err != nil {
		return err
	}
	people := []types.JID{person}
	if redeemed.amount > 0 {
		text = fmt.Sprintf("🏦 @%s resgatou a poupança: voltou %s, com %s de juros%s\n\n%s",
			person.User, coins(redeemed.amount+redeemed.interest-tax), coins(redeemed.interest-tax), savingsTaxText(tax, levy), text)
	}
	if tax > 0 {
		people = append(people, levy.president.jid)
	}
	_, err = b.sendMentions(ctx, chat, text, people)
	return err
}

func savingsTaxText(tax int64, levy pixLevy) string {
	if tax == 0 {
		return ""
	}
	return fmt.Sprintf(" (já tirando %s, %d%% dos juros, de imposto pro bolso de @%s, por decreto)", coins(tax), levy.percent, levy.president.jid.User)
}

func (b *bot) savingsReply(ctx context.Context, evt *events.Message, acc account, now time.Time) (string, error) {
	chat := evt.Info.Chat.ToNonAD()
	user := acc.jid.User
	current, found, err := b.wallets.savings(ctx, chat, acc)
	if err != nil {
		return "", fmt.Errorf("ler poupança: %w", err)
	}
	if found {
		return lockedSavingsText(user, current, now), nil
	}

	job, err := b.professionOf(ctx, chat, acc)
	if err != nil {
		return "", err
	}
	rate := max(int64(savingsRate)+job.interest+job.decrees.level(interestDecree), 0)
	usage := fmt.Sprintf("🏦 Pra guardar moedas: !poupar 100 (ou metade, 10%%, tudo). Elas ficam travadas por %s e voltam com %d%% de juros, até %s por vez",
		formatWait(savingsLock), rate, coins(savingsCap)) + job.decrees.note(interestDecree) + job.decrees.note(savingsTaxDecree)
	arg := firstPlainArg(messageText(evt.Message))
	if arg == "" {
		return usage, nil
	}
	balance, _, err := b.wallets.balance(ctx, chat, acc)
	if err != nil {
		return "", fmt.Errorf("ler saldo: %w", err)
	}
	if balance == 0 {
		return "Você tá liso 🪹 Usa !daily e !trabalhar pra ter o que guardar", nil
	}
	amount, ok := parseStake(arg, balance)
	if !ok {
		return usage, nil
	}
	if amount > balance {
		return "Você só tem " + coins(balance), nil
	}
	amount = min(amount, savingsCap)
	if amount < savingsMin {
		return "O mínimo pra poupar é " + coins(savingsMin), nil
	}

	deposit := savings{amount: amount, interest: amount * rate / 100, due: now.Add(savingsLock)}
	err = b.wallets.deposit(ctx, chat, acc, deposit)
	if errors.Is(err, errNoFunds) {
		return "Seu saldo mudou no meio do depósito e não dá mais 🤨 Confere no !saldo", nil
	}
	if errors.Is(err, errSavingsActive) {
		return "Você já tem uma poupança rodando, espera ela liberar", nil
	}
	if err != nil {
		return "", fmt.Errorf("guardar na poupança: %w", err)
	}
	return fmt.Sprintf("🔒 @%s guardou %s na poupança. Daqui a %s volta com %s de juros, é só mandar !poupar pra resgatar",
		user, coins(deposit.amount), formatWait(savingsLock), coins(deposit.interest)) + job.decrees.note(interestDecree) + job.decrees.note(savingsTaxDecree), nil
}

func lockedSavingsText(user string, sv savings, now time.Time) string {
	return fmt.Sprintf("🔒 @%s tem %s na poupança, rendendo %s de juros. Libera em %s",
		user, coins(sv.amount), coins(sv.interest), formatWait(sv.due.Sub(now)))
}
