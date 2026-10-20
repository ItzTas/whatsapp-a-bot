package economy

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	reportChance = 0.35
)

var reportCooldown = cooldown{column: "denuncia_em", every: 15 * time.Minute}

type snitchFate int

const (
	snitchJailed snitchFate = iota
	snitchKilled
	snitchCaptured
	snitchFates
)

type payback struct {
	fate    snitchFate
	left    time.Duration
	fine    int64
	lost    int64
	freed   types.JID
	deposed bool
}

func (b *bot) denunciar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	aim, problem := b.targetOf(ctx, evt)
	if problem != "" {
		return b.SendText(ctx, chat, problem)
	}
	if !aim.found() {
		return b.SendText(ctx, chat, "Marca quem você vai denunciar: !denunciar @pessoa\n"+targetHint)
	}
	reporter, suspect := evt.Info.Sender.ToNonAD(), aim.person
	reporterAcc, suspectAcc := b.Account(ctx, reporter), b.Account(ctx, suspect)
	if reporterAcc.Key == suspectAcc.Key {
		return b.SendText(ctx, chat, "Quer se denunciar? Vai direto na delegacia e confessa 🤡")
	}

	reporterJob, suspectJob, err := b.rivalProfessions(ctx, chat, reporterAcc, suspectAcc)
	if err != nil {
		return err
	}
	if reporterJob.noSnitch && !reporterJob.president {
		return b.punishSnitch(ctx, chat, reporterAcc, suspectAcc, reporterJob, reporter, suspect)
	}
	sentence := Profession{jailTime: baseJail}.sentence(suspectJob)
	a, err := b.Wallets.arrest(ctx, chat, reporterAcc, suspectAcc, cooldown{column: reportCooldown.column, every: reporterJob.hasted(reportCooldown.every)}, nil, sentence, arrestOdds{chance: reporterJob.sober(reporterJob.reportOddsOn(suspectJob)), immunity: suspectJob.immunity()}, false, false, time.Now())
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

func (b *bot) punishSnitch(ctx context.Context, chat types.JID, snitchAcc, suspectAcc core.Account, job Profession, snitch, suspect types.JID) error {
	p, err := b.Wallets.punishSnitch(ctx, chat, snitchAcc, suspectAcc, job.snitchSentence(), time.Now())
	if err != nil {
		return fmt.Errorf("castigar x9: %w", err)
	}
	rescue, rescued := arrest{freed: p.freed}.rescue()
	_, err = b.SendMentions(ctx, chat, p.text(job, snitch, suspect)+rescue, append([]types.JID{snitch, suspect}, rescued...))
	return err
}

func (p payback) text(job Profession, snitch, suspect types.JID) string {
	caught := fmt.Sprintf("🐀 @%s %s e tentou dar uma de X-9 contra @%s. A quadrilha descobriu", snitch.User, job.title, suspect.User)
	endings := map[snitchFate]func() string{
		snitchJailed: func() string {
			return fmt.Sprintf(", entregou pra polícia e agora @%s pegou %s de cadeia (sai em %s)", snitch.User, core.FormatWait(job.snitchSentence()), core.FormatWait(p.left)) + fineText(p.fine)
		},
		snitchKilled: func() string {
			text := fmt.Sprintf(" e fez queima de arquivo: @%s morreu e fica %s sem comando de economia, a não ser que pague o !hospital ou alguém consiga !reviver", snitch.User, core.FormatWait(p.left))
			if funeral := funeralText(snitch, p.lost); funeral != "" {
				text += "\n" + funeral
			}
			if p.deposed {
				text += "\n" + deposedText(snitch)
			}
			return text
		},
		snitchCaptured: func() string {
			return fmt.Sprintf(", amarrou o X-9 e entregou de presente pra @%s: @%s fica %s em cativeiro, a não ser que consiga !fugir, que a polícia !prender @%s ou que alguém pague o !resgate", suspect.User, snitch.User, core.FormatWait(p.left), suspect.User)
		},
	}
	return caught + endings[p.fate]()
}

func (p Profession) snitchSentence() time.Duration {
	return Profession{jailTime: baseJail}.sentence(p)
}

func (p Profession) reportOdds() float64 {
	return clampChance(reportChance + p.reportBonus)
}

func (p Profession) reportOddsOn(suspect Profession) float64 {
	return clampChance(reportChance + p.reportBonus + suspect.celebrity().snitch)
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
	if a.immune {
		return immunityText(reporter, suspect)
	}
	if a.dismissed {
		return fmt.Sprintf("📵 @%s ligou pro 190 pra denunciar @%s, mas acharam que era trote e ninguém apareceu", reporter.User, suspect.User) + reportEdgeText(reporterJob, suspectJob, suspect)
	}
	return fmt.Sprintf("📞 @%s ligou pro 190 e dedurou @%s pelo crime de %s atrás! A viatura chegou e levou: fica %s na cadeia, sem poder apostar, roubar nem furtar", reporter.User, suspect.User, core.FormatWait(a.crimeAgo), core.FormatWait(sentence)) +
		jailFactorText(suspect, suspectJob) + fineText(a.fine) + seizedText(a.seized) + seizedImplantsText(a.unplugged) + reportEdgeText(reporterJob, suspectJob, suspect)
}

func reportEdgeText(job, suspectJob Profession, suspect types.JID) string {
	odds, spotted := percent(job.reportOddsOn(suspectJob)), spottedText(suspectJob, suspect.User)
	if job.reportBonus > 0 {
		return fmt.Sprintf("\n🕵️ Como %s, a denúncia tinha %d%% de chance", job.title, odds) + spotted
	}
	if job.reportBonus < 0 {
		return fmt.Sprintf("\n🙄 Como %s, a denúncia tinha só %d%% de chance", job.title, odds) + spotted
	}
	return spotted
}

func (s *walletStore) punishSnitch(ctx context.Context, chat types.JID, snitch, suspect core.Account, jail time.Duration, now time.Time) (p payback, err error) {
	fate := snitchFate(rand.IntN(int(snitchFates)))
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, snitch); err != nil {
			return err
		}
		punishments := map[snitchFate]func() (payback, error){
			snitchJailed:   func() (payback, error) { return jailSnitch(ctx, tx, chat, snitch, jail, now) },
			snitchKilled:   func() (payback, error) { return killSnitch(ctx, tx, chat, snitch, now) },
			snitchCaptured: func() (payback, error) { return captureSnitch(ctx, tx, chat, snitch, suspect, jail, now) },
		}
		p, err = punishments[fate]()
		return err
	})
	return p, err
}

func jailSnitch(ctx context.Context, tx *ledgerTx, chat types.JID, snitch core.Account, jail time.Duration, now time.Time) (p payback, err error) {
	var until int64
	err = tx.QueryRowContext(ctx, "SELECT preso_ate FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), snitch.Key).Scan(&until)
	if err != nil {
		return p, err
	}
	release := time.Unix(max(until, now.Unix()), 0).Add(jail)
	p = payback{fate: snitchJailed, left: release.Sub(now)}
	if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET preso_ate = ? WHERE chat = ? AND pessoa = ?", release.Unix(), chat.String(), snitch.Key); err != nil {
		return p, err
	}
	if p.fine, err = lockUp(ctx, tx, chat, snitch, now); err != nil {
		return p, err
	}
	p.freed, err = freeHostage(ctx, tx, chat, snitch, now)
	return p, err
}

func killSnitch(ctx context.Context, tx *ledgerTx, chat types.JID, snitch core.Account, now time.Time) (payback, error) {
	d, err := die(ctx, tx, chat, snitch, now)
	return payback{fate: snitchKilled, left: d.hold, lost: d.lost, freed: d.freed, deposed: d.deposed}, err
}

func captureSnitch(ctx context.Context, tx *ledgerTx, chat types.JID, snitch, captor core.Account, jail time.Duration, now time.Time) (payback, error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM sequestros WHERE chat = ? AND ate <= ?", chat.String(), now.Unix()); err != nil {
		return payback{}, err
	}
	res, err := tx.ExecContext(ctx,
		"INSERT INTO sequestros (chat, refem, refem_jid, sequestrador, ate) VALUES (?, ?, ?, ?, ?) ON CONFLICT DO NOTHING",
		chat.String(), snitch.Key, snitch.JID.String(), captor.Key, now.Add(hostageHold).Unix(),
	)
	if err != nil {
		return payback{}, err
	}
	taken, err := res.RowsAffected()
	if err != nil {
		return payback{}, err
	}
	if taken == 0 {
		return jailSnitch(ctx, tx, chat, snitch, jail, now)
	}
	freed, err := freeHostage(ctx, tx, chat, snitch, now)
	return payback{fate: snitchCaptured, left: hostageHold, freed: freed}, err
}
