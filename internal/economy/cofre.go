package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

var errVaultChanged = errors.New("cofre ou saldo mudou")

type homeVault struct {
	balance int64
	stored  int64
	home    estate
	housed  bool
}

func (v homeVault) free() int64 {
	return max(v.home.vault-v.stored, 0)
}

func (v homeVault) gauge() string {
	return fmt.Sprintf("🔐 Cofre: %s de %s", Coins(v.stored), Coins(v.home.vault))
}

func vaultArg(text string) string {
	fields := strings.Fields(core.FoldLetters(text))
	if len(fields) < 3 {
		return ""
	}
	return fields[2]
}

const homelessVaultText = "🪹 Esconder dinheiro onde, se você não tem nem onde morar? Compra um imóvel no !imovel primeiro"

func (b *bot) fillVault(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	v, err := b.Wallets.vault(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler cofre: %w", err)
	}
	if !v.housed {
		return b.SendText(ctx, chat, homelessVaultText)
	}
	if v.free() == 0 {
		return b.SendText(ctx, chat, fmt.Sprintf("🔐 Seu cofre (%s) já tá cheio, com %s. Sobe de imóvel no !imovel pra caber mais", v.home.label(), Coins(v.stored)))
	}
	if v.balance == 0 {
		return b.SendText(ctx, chat, "Você tá liso 🪹 Não tem nada pra esconder")
	}
	asked, ok := ParseStake(vaultArg(core.MessageText(evt.Message)), v.balance)
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz quanto quer esconder: !imovel depositar 500 (ou metade, 10%%, tudo). Ainda cabe %s no cofre", Coins(v.free())))
	}
	if asked > v.balance {
		return b.SendText(ctx, chat, "Você só tem "+Coins(v.balance))
	}
	amount := min(asked, v.free())
	v.balance, v.stored, err = b.Wallets.shiftVault(ctx, chat, acc, amount, v.home.vault)
	if errors.Is(err, errVaultChanged) {
		return b.SendText(ctx, chat, "Seu saldo mudou no meio do caminho e não dá mais 🤨 Confere no !saldo")
	}
	if err != nil {
		return fmt.Errorf("guardar no cofre: %w", err)
	}
	overflow := ""
	if amount < asked {
		overflow = fmt.Sprintf(" (só cabia isso, os outros %s continuam no saldo)", Coins(asked-amount))
	}
	text := fmt.Sprintf("🔐 @%s escondeu %s no cofre de casa%s. Lá ninguém rouba\n💰 Saldo: %s\n%s",
		person.User, Coins(amount), overflow, Coins(v.balance), v.gauge())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) drainVault(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	v, err := b.Wallets.vault(ctx, chat, acc)
	if err != nil {
		return fmt.Errorf("ler cofre: %w", err)
	}
	if v.stored == 0 {
		return b.SendText(ctx, chat, "🔐 Seu cofre tá vazio. Manda !imovel depositar <valor> pra esconder moedas em casa")
	}
	amount, ok := ParseStake(vaultArg(core.MessageText(evt.Message)), v.stored)
	if !ok {
		return b.SendText(ctx, chat, fmt.Sprintf("Diz quanto quer sacar: !imovel sacar 500 (ou metade, 10%%, tudo). Tem %s no cofre", Coins(v.stored)))
	}
	if amount > v.stored {
		return b.SendText(ctx, chat, "Você só tem "+Coins(v.stored)+" no cofre")
	}
	v.balance, v.stored, err = b.Wallets.shiftVault(ctx, chat, acc, -amount, v.home.vault)
	if errors.Is(err, errVaultChanged) {
		return b.SendText(ctx, chat, "O cofre mudou no meio do caminho 🤨 Confere no !imovel cofre")
	}
	if err != nil {
		return fmt.Errorf("sacar do cofre: %w", err)
	}
	text := fmt.Sprintf("💸 @%s tirou %s do cofre de casa. Agora tá no saldo, ao alcance de qualquer ladrão\n💰 Saldo: %s\n%s",
		person.User, Coins(amount), Coins(v.balance), v.gauge())
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	return err
}

func (b *bot) showVault(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	v, err := b.Wallets.vault(ctx, chat, b.Account(ctx, person))
	if err != nil {
		return fmt.Errorf("ler cofre: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, vaultText(person, v), []types.JID{person})
	return err
}

func vaultText(person types.JID, v homeVault) string {
	if !v.housed {
		return fmt.Sprintf("🪹 @%s não tem imóvel, então não tem cofre. Compra um no !imovel", person.User)
	}
	return fmt.Sprintf("🔐 *Cofre da casa de @%s* (%s)\n%s, cabe mais %s\n\n_Moeda no cofre não é roubada, não paga multa nem vai embora no enterro, mas também não serve pra nada até voltar pro saldo. Manda !imovel depositar <valor> pra esconder e !imovel sacar <valor> pra pegar de volta_",
		person.User, v.home.label(), v.gauge(), Coins(v.free()))
}

func (s *walletStore) vault(ctx context.Context, chat types.JID, acc core.Account) (v homeVault, err error) {
	var level int
	err = s.db.QueryRowContext(ctx, "SELECT saldo, cofre_casa, imovel FROM carteiras WHERE chat = ? AND pessoa = ?",
		chat.String(), acc.Key).Scan(&v.balance, &v.stored, &level)
	if errors.Is(err, sql.ErrNoRows) {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	v.home, v.housed = estateAt(level)
	return v, nil
}

func (s *walletStore) shiftVault(ctx context.Context, chat types.JID, acc core.Account, amount, capacity int64) (balance, stored int64, err error) {
	err = s.db.QueryRowContext(ctx,
		`UPDATE carteiras SET saldo = saldo - ?, cofre_casa = cofre_casa + ?
		 WHERE chat = ? AND pessoa = ? AND saldo >= ? AND cofre_casa + ? BETWEEN 0 AND ?
		 RETURNING saldo, cofre_casa`,
		amount, amount, chat.String(), acc.Key, amount, amount, capacity,
	).Scan(&balance, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, errVaultChanged
	}
	return balance, stored, err
}
