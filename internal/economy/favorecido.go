package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const (
	protegePayBonus   = 15
	protegeDailyBonus = 15
	protegeGuard      = 0.05
	protegeSyntax     = "!decreto favorecer @pessoa"
	protegeOf         = `EXISTS (SELECT 1 FROM ditadura_favoritos f JOIN presidencias p ON p.chat = f.chat
		 WHERE f.chat = c.chat AND f.pessoa = c.pessoa AND p.pessoa != c.pessoa AND p.mandato_ate > ? AND p.ditadura != 0 AND f.favorito_em >= p.ditadura)`
)

var protegeNames = []string{"favorecer", "favorito", "favorita", "apadrinhar", "afilhado", "afilhada", "puxasaco"}

func (p Profession) patronized(protege bool) Profession {
	if !protege {
		return p
	}
	p.protege = true
	p.min += p.min * protegePayBonus / 100
	p.max += p.max * protegePayBonus / 100
	p.dailyBonus += protegeDailyBonus
	p.cordon += protegeGuard
	return p
}

func protegePerk() string {
	return fmt.Sprintf("🎖️ É o favorito do ditador: o !trabalhar paga +%d%%, o !daily rende +%d%% e quem tenta !sequestrar ou !matar você tem mais %d%% de chance de errar", protegePayBonus, protegeDailyBonus, percent(protegeGuard))
}

func protegeHelp() core.Usage {
	return core.Usage{protegeSyntax, fmt.Sprintf("só na ditadura: escolhe o favorito do regime, só um de cada vez (escolher outro tira o anterior). O favorito ganha +%d%% no !trabalhar, +%d%% no !daily e +%d%% de segurança contra !matar e !sequestrar, e perde tudo quando a ditadura cai",
		protegePayBonus, protegeDailyBonus, percent(protegeGuard))}
}

func (b *bot) protegeDecree(ctx context.Context, call decreeCall) (string, error) {
	if !call.p.decrees.on(tyrannyDecree) {
		return "🎖️ Ter favorito é coisa de ditador. Pra escolher um, primeiro manda !decreto ditadura", nil
	}
	if len(call.targets) == 0 {
		return "🎖️ Marca quem vai ser o favorito do regime: " + protegeSyntax, nil
	}
	user := call.p.president.JID.User
	protege := call.targets[0]
	if protege.Key == call.p.president.Key {
		return fmt.Sprintf("🤨 @%s já é o ditador, favorito maior não existe. Escolhe outra pessoa", user), nil
	}
	if b.IsSelf(protege.JID) {
		return "🤖 Eu não aceito boquinha de ditador. Escolhe outra pessoa", nil
	}
	swap, err := b.Wallets.anointProtege(ctx, call.chat, call.p.president, protege, call.now)
	if err != nil {
		return "", fmt.Errorf("decreto favorecer: %w", err)
	}
	switch swap {
	case protegeOver:
		return termOverText, nil
	case protegeSame:
		return fmt.Sprintf("🎖️ @%s já é o favorito de @%s", protege.JID.User, user), nil
	}
	text := fmt.Sprintf("🎖️ *Favorito do regime!* @%s escolheu @%s pra ser o queridinho da ditadura: +%d%% no !trabalhar, +%d%% no !daily e +%d%% de segurança contra !matar e !sequestrar enquanto a ditadura durar",
		user, protege.JID.User, protegePayBonus, protegeDailyBonus, percent(protegeGuard))
	if swap == protegeReplaced {
		text += "\n_O favorito de antes perdeu a boquinha_"
	}
	return text, nil
}

type protegeSwap int

const (
	protegeNew protegeSwap = iota
	protegeReplaced
	protegeSame
	protegeOver
)

func (s *walletStore) anointProtege(ctx context.Context, chat types.JID, president, protege core.Account, now time.Time) (swap protegeSwap, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var since int64
		err := tx.QueryRowContext(ctx,
			"SELECT "+tyrannyDecree.column+" FROM presidencias WHERE chat = ? AND pessoa = ? AND mandato_ate > ? AND "+tyrannyDecree.column+" != 0",
			chat.String(), president.Key, now.Unix(),
		).Scan(&since)
		if err != nil {
			return err
		}
		var current string
		err = tx.QueryRowContext(ctx,
			"SELECT COALESCE((SELECT pessoa FROM ditadura_favoritos WHERE chat = ? AND favorito_em >= ?), '')",
			chat.String(), since,
		).Scan(&current)
		if err != nil {
			return err
		}
		if current == protege.Key {
			swap = protegeSame
			return nil
		}
		if current != "" {
			swap = protegeReplaced
		}
		if err := openWallet(ctx, tx, chat, protege); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO ditadura_favoritos (chat, pessoa, jid, favorito_em) VALUES (?, ?, ?, ?)
			 ON CONFLICT (chat) DO UPDATE SET pessoa = excluded.pessoa, jid = excluded.jid, favorito_em = excluded.favorito_em`,
			chat.String(), protege.Key, protege.JID.String(), now.Unix(),
		)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return protegeOver, nil
	}
	return swap, err
}
