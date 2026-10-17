package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	reportChance = 0.35
)

var reportCooldown = cooldown{column: "denuncia_em", every: 15 * time.Minute}

func (b *bot) denunciar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := core.MentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.SendText(ctx, chat, "Marca quem você vai denunciar: !denunciar @pessoa")
	}
	reporter, suspect := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	reporterAcc, suspectAcc := b.Account(ctx, reporter), b.Account(ctx, suspect)
	if reporterAcc.Key == suspectAcc.Key {
		return b.SendText(ctx, chat, "Quer se denunciar? Vai direto na delegacia e confessa 🤡")
	}

	reporterJob, suspectJob, err := b.rivalProfessions(ctx, chat, reporterAcc, suspectAcc)
	if err != nil {
		return err
	}
	if shield := arrestShield(suspect, suspectJob); shield != "" {
		_, err = b.SendMentions(ctx, chat, shield, []types.JID{suspect})
		return err
	}
	if reporterJob.noSnitch && !reporterJob.president {
		return b.jailSnitch(ctx, chat, reporterAcc, reporterJob, reporter, suspect)
	}
	sentence := Profession{jailTime: baseJail}.sentence(suspectJob)
	a, err := b.Wallets.arrest(ctx, chat, reporterAcc, suspectAcc, cooldown{column: reportCooldown.column, every: reporterJob.hasted(reportCooldown.every)}, nil, sentence, reporterJob.reportOdds(), false, false, time.Now())
	if err != nil {
		return fmt.Errorf("denunciar: %w", err)
	}
	if a.clean {
		if err := b.Wallets.fileGrievance(ctx, chat, suspectAcc, reporterAcc, reportGrievance, time.Now()); err != nil {
			return fmt.Errorf("guardar a queixa da denúncia: %w", err)
		}
	}
	rescue, freed := a.rescue()
	_, err = b.SendMentions(ctx, chat, reportText(a, reporterJob, suspectJob, sentence, reporter, suspect)+rescue, append([]types.JID{reporter, suspect}, freed...))
	return err
}

func (b *bot) jailSnitch(ctx context.Context, chat types.JID, snitchAcc core.Account, job Profession, snitch, suspect types.JID) error {
	jail := job.snitchSentence()
	left, fine, freed, err := b.Wallets.jailSnitch(ctx, chat, snitchAcc, jail, time.Now())
	if err != nil {
		return fmt.Errorf("prender x9: %w", err)
	}
	rescue, rescued := arrest{freed: freed}.rescue()
	text := fmt.Sprintf("🐀 @%s %s e tentou dar uma de X-9 contra @%s. A quadrilha descobriu, entregou pra polícia e agora @%s pegou %s de cadeia (sai em %s)",
		snitch.User, job.title, suspect.User, snitch.User, core.FormatWait(jail), core.FormatWait(left))
	_, err = b.SendMentions(ctx, chat, text+fineText(fine)+rescue, append([]types.JID{snitch, suspect}, rescued...))
	return err
}

func (p Profession) snitchSentence() time.Duration {
	return Profession{jailTime: baseJail}.sentence(p)
}

func (p Profession) reportOdds() float64 {
	return clampChance(reportChance + p.reportBonus)
}

func reportText(a arrest, reporterJob, suspectJob Profession, sentence time.Duration, reporter, suspect types.JID) string {
	if a.held > 0 {
		return fmt.Sprintf("🔒 @%s já tá na cadeia. Sai em %s", suspect.User, core.FormatWait(a.held))
	}
	if a.clean {
		return fmt.Sprintf("🤷 @%s tá com a ficha limpa. Só dá pra denunciar quem tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto, negociou no !mercadonegro ou desviou o cofre da igreja nas últimas %s", suspect.User, core.FormatWait(crimeWindow)) + grievanceNote(suspect, reporter)
	}
	if a.wait > 0 {
		return fmt.Sprintf("☎️ A polícia ainda tá ignorando a última ligação de @%s. Espera %s pra denunciar de novo", reporter.User, core.FormatWait(a.wait))
	}
	if a.dismissed {
		return fmt.Sprintf("📵 @%s ligou pro 190 pra denunciar @%s, mas acharam que era trote e ninguém apareceu", reporter.User, suspect.User) + reportEdgeText(reporterJob)
	}
	return fmt.Sprintf("📞 @%s ligou pro 190 e dedurou @%s pelo crime de %s atrás! A viatura chegou e levou: fica %s na cadeia, sem poder apostar, roubar nem furtar", reporter.User, suspect.User, core.FormatWait(a.crimeAgo), core.FormatWait(sentence)) +
		jailFactorText(suspect, suspectJob) + fineText(a.fine) + seizedText(a.seized) + reportEdgeText(reporterJob)
}

func reportEdgeText(job Profession) string {
	if job.reportBonus > 0 {
		return fmt.Sprintf("\n🕵️ Como %s, a denúncia tinha %d%% de chance", job.title, percent(job.reportOdds()))
	}
	if job.reportBonus < 0 {
		return fmt.Sprintf("\n🙄 Como %s, a denúncia tinha só %d%% de chance", job.title, percent(job.reportOdds()))
	}
	return ""
}

func (s *walletStore) jailSnitch(ctx context.Context, chat types.JID, snitch core.Account, jail time.Duration, now time.Time) (left time.Duration, fine int64, freed types.JID, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, snitch); err != nil {
			return err
		}
		var until int64
		err := tx.QueryRowContext(ctx, "SELECT preso_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), snitch.Key).Scan(&until)
		if err != nil {
			return err
		}
		release := time.Unix(max(until, now.Unix()), 0).Add(jail)
		left = release.Sub(now)
		if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = ? WHERE chat = ? AND pessoa = ?", release.Unix(), chat.String(), snitch.Key); err != nil {
			return err
		}
		if fine, err = lockUp(ctx, tx, chat, snitch, now); err != nil {
			return err
		}
		freed, err = freeHostage(ctx, tx, chat, snitch, now)
		return err
	})
	return left, fine, freed, err
}
