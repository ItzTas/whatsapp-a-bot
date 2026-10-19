package economy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) suicidio(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	d, err := b.Wallets.suicide(ctx, chat, b.Account(ctx, person), time.Now())
	if err != nil {
		return fmt.Errorf("suicidio: %w", err)
	}
	text, freed := farewellText(d, person)
	_, err = b.SendMentions(ctx, chat, text, append([]types.JID{person}, freed...))
	return err
}

func (s *walletStore) suicide(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (d death, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		d, err = die(ctx, tx, chat, acc, now)
		return err
	})
	return d, err
}

func farewellText(d death, person types.JID) (string, []types.JID) {
	scenes := []string{
		"🪦 @%s viu o boleto do mês e decidiu que preferia o caixão",
		"👻 @%s saiu da vida igual sai de grupo de família: sem avisar ninguém",
		"🕊️ @%s deu tchau pra galera e foi descansar mais cedo",
		"📉 @%s olhou o !ricos, não se achou em lugar nenhum e desistiu de tudo",
		"🛌 @%s cansou do !trabalhar e pediu as contas da vida",
	}
	lines := []string{
		fmt.Sprintf(core.PickOne(scenes), person.User),
		"",
		fmt.Sprintf("⚰️ @%s tá morto(a) por %s: até lá, nada de comando de economia. Dá pra voltar antes pagando o !hospital ou com alguém que saiba !reviver", person.User, core.FormatWait(d.hold)),
	}
	if funeral := funeralText(person, d.lost); funeral != "" {
		lines = append(lines, funeral)
	}
	if d.deposed {
		lines = append(lines, deposedText(person))
	}
	escape, freed := hostageEscape(d.freed, "🔓 O refém de @"+person.User+" aproveitou o velório pra fugir: @%s tá livre!")
	if escape != "" {
		lines = append(lines, escape)
	}
	lines = append(lines, "", "💛 Isso aqui é só o jogo. Se a vida real tiver pesada, o CVV atende no 188, a qualquer hora e de graça")
	return strings.Join(lines, "\n"), freed
}
