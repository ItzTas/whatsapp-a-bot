package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func (s *walletStore) numberOf(ctx context.Context, chat types.JID, acc core.Account) (number int, err error) {
	err = s.db.QueryRowContext(ctx, "SELECT numero FROM numeros WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&number)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return number, err
}

func (s *walletStore) numbered(ctx context.Context, chat types.JID, number int) (person types.JID, found bool, err error) {
	var jid string
	err = s.db.QueryRowContext(ctx,
		`SELECT c.jid FROM numeros n JOIN carteiras c ON c.chat = n.chat AND c.pessoa = n.pessoa
		 WHERE n.chat = ? AND n.numero = ?`,
		chat.String(), number,
	).Scan(&jid)
	if errors.Is(err, sql.ErrNoRows) {
		return types.JID{}, false, nil
	}
	if err != nil {
		return types.JID{}, false, err
	}
	person, err = types.ParseJID(jid)
	return person.ToNonAD(), err == nil, err
}

func numberTag(number int) string {
	if number == 0 {
		return ""
	}
	return fmt.Sprintf(" (#%d)", number)
}

func (b *bot) personNumbered(ctx context.Context, chat types.JID, text string) (aim target, problem string, typed bool) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return target{}, "", false
	}
	digits, hashed := strings.CutPrefix(fields[1], "#")
	number, err := strconv.Atoi(digits)
	if !hashed || err != nil || number <= 0 {
		return target{}, "", false
	}
	person, found, err := b.Wallets.numbered(ctx, chat, number)
	if err != nil {
		b.Client.Log.Warnf("achar o #%d em %s: %v", number, chat, err)
		return target{}, fmt.Sprintf("Não consegui procurar o #%d agora 😵 Tenta de novo daqui a pouco", number), true
	}
	if !found {
		return target{}, fmt.Sprintf("Ninguém tem o número #%d nessa economia 🤷 O número de cada um aparece no !saldo e no !ricos", number), true
	}
	return target{person: person, rest: strings.Join(slices.Concat(fields[:1], fields[2:]), " ")}, "", true
}
