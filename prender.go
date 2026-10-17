package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	crimeWindow    = 2 * time.Hour
	baseJail       = 10 * time.Minute
	fineMinPercent = 1
	fineMaxPercent = 3
)

type abuseKind struct {
	name    string
	column  string
	charges string
	every   time.Duration
	perk    string
}

var (
	arrestAbuse = abuseKind{
		name: "prisão", column: "abuso_em", charges: "abusos", every: 18 * time.Hour,
		perk: "se ainda tá na papelada da última prisão, !prender de novo na hora",
	}
	workAbuse = abuseKind{
		name: "trabalho", column: "abuso_trabalho_em", charges: "abusos_trabalho", every: 6 * time.Hour,
		perk: "se ainda tá cansado do último trampo, !trabalhar de novo na hora",
	}
	robAbuse = abuseKind{
		name: "roubo", column: "abuso_roubo_em", charges: "abusos_roubo", every: 6 * time.Hour,
		perk: "se a polícia ainda tá procurando, !roubar de novo na hora",
	}
	kidnapAbuse = abuseKind{
		name: "sequestro", column: "abuso_sequestro_em", charges: "abusos_sequestro", every: 12 * time.Hour,
		perk: "se ainda não deu o tempo do último sequestro, !sequestrar de novo na hora",
	}
	abuseKinds = []abuseKind{arrestAbuse, workAbuse, robAbuse, kidnapAbuse}
)

type arrest struct {
	wait      time.Duration
	held      time.Duration
	clean     bool
	dismissed bool
	abuse     powerAbuse
	crimeAgo  time.Duration
	freed     types.JID
	fine      int64
	seized    []stack
}

func (b *bot) prender(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.sendText(ctx, chat, "Marca quem você vai prender: !prender @pessoa")
	}
	cop, suspect := evt.Info.Sender.ToNonAD(), mentions[0].ToNonAD()
	copAcc, suspectAcc := b.account(ctx, cop), b.account(ctx, suspect)
	if copAcc.key == suspectAcc.key {
		return b.sendText(ctx, chat, "Quer se entregar? Pede pra outro policial te prender 🤡")
	}

	copJob, suspectJob, owed, err := b.rivalsWithDebt(ctx, chat, copAcc, suspectAcc)
	if err != nil {
		return err
	}
	if copJob.jailTime == 0 && owed == nil {
		return b.sendText(ctx, chat, "🚫 Só quem é da polícia ou o presidente pode prender: "+arrestingProfessions()+". Ou quem emprestou moedas com o !agiota pra alguém que passou do prazo")
	}
	if shield := arrestShield(suspect, suspectJob); shield != "" {
		_, err = b.sendMentions(ctx, chat, shield, []types.JID{suspect})
		return err
	}
	sentence, every := copJob.sentence(suspectJob), copJob.arrestEvery
	if copJob.jailTime == 0 {
		sentence, every = profession{jailTime: baseJail}.sentence(suspectJob), copJob.collectionEvery()
	}
	a, err := b.wallets.arrest(ctx, chat, copAcc, suspectAcc, cooldown{column: "prisao_em", every: copJob.hasted(every)}, copJob.abuse(arrestAbuse), sentence, 1, owed != nil, true, time.Now())
	if err != nil {
		return fmt.Errorf("prender: %w", err)
	}
	if a.clean {
		if err := b.wallets.fileGrievance(ctx, chat, suspectAcc, copAcc, arrestGrievance, time.Now()); err != nil {
			return fmt.Errorf("guardar a queixa da prisão: %w", err)
		}
	}
	text := arrestText(a, copJob, suspectJob, sentence, cop, suspect)
	if owed != nil {
		text = debtArrestText(a, copJob, suspectJob, sentence, owed)
	}
	rescue, freed := a.rescue()
	_, err = b.sendMentions(ctx, chat, text+seizedText(a.seized)+rescue, append([]types.JID{cop, suspect}, freed...))
	return err
}

func debtArrestText(a arrest, copJob, suspectJob profession, sentence time.Duration, owed *debt) string {
	cop, suspect := owed.lender.jid, owed.debtor.jid
	if a.held > 0 || a.wait > 0 {
		return arrestText(a, copJob, suspectJob, sentence, cop, suspect)
	}
	return fmt.Sprintf("⚖️ @%s cansou de esperar e mandou prender @%s por calote! Fica %s na cadeia, sem poder apostar, roubar nem furtar", cop.User, suspect.User, formatWait(sentence)) +
		a.abuse.note("Abusou do poder e pulou a papelada") + jailFactorText(suspect, suspectJob) + paidFineText(a.fine, cop) + debtCollectionText(owed)
}

type powerAbuse struct {
	kind         abuseKind
	used         bool
	presidential bool
	left         int
}

type abuser func(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) (powerAbuse, error)

func (p profession) abuse(kind abuseKind) abuser {
	own := slices.Contains(p.abuses, kind)
	if !p.president && !own {
		return nil
	}
	return func(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) (powerAbuse, error) {
		if p.president {
			pa, err := presidentialAbuse(ctx, tx, chat, acc, kind, now)
			if err != nil || pa.used {
				return pa, err
			}
		}
		if !own {
			return powerAbuse{}, nil
		}
		wait, err := stampCooldown(ctx, tx, chat, acc, cooldown{column: kind.column, every: kind.every}, now)
		return powerAbuse{kind: kind, used: err == nil && wait == 0}, err
	}
}

func (p profession) abusesText() string {
	lines := make([]string, len(p.abuses))
	for i, kind := range p.abuses {
		lines[i] = fmt.Sprintf("😈 A cada %s pode abusar do poder: %s", formatWait(kind.every), kind.perk)
	}
	return strings.Join(lines, "\n")
}

func (p profession) sentence(suspect profession) time.Duration {
	jail := p.jailTime * time.Duration(max(100+suspect.jailFactor, 0)) / 100
	return jail * time.Duration(100+suspect.decrees.level(sentenceDecree)) / 100
}

func arrestText(a arrest, copJob, suspectJob profession, sentence time.Duration, cop, suspect types.JID) string {
	if a.held > 0 {
		return fmt.Sprintf("🔒 @%s já tá na cadeia. Sai em %s", suspect.User, formatWait(a.held))
	}
	if a.clean {
		return fmt.Sprintf("🤷 @%s tá com a ficha limpa. Só dá pra prender quem tentou !roubar, !furtar, !sequestrar, !matar, entrou num !assalto ou negociou no !mercadonegro nas últimas %s", suspect.User, formatWait(crimeWindow)) + grievanceNote(suspect, cop)
	}
	if a.wait > 0 {
		return fmt.Sprintf("📝 @%s ainda tá preenchendo a papelada da última prisão. Espera %s pra prender de novo", cop.User, formatWait(a.wait))
	}
	if a.abuse.used {
		return fmt.Sprintf("😈 %s @%s abusou do poder, pulou a papelada e prendeu @%s pelo crime de %s atrás! Fica %s na cadeia, sem poder apostar, roubar nem furtar. %s", copJob.badge(), cop.User, suspect.User, formatWait(a.crimeAgo), formatWait(sentence), a.abuse.nextText()) + jailFactorText(suspect, suspectJob) + paidFineText(a.fine, cop)
	}
	return fmt.Sprintf("🚔 %s @%s prendeu @%s pelo crime de %s atrás! Fica %s na cadeia, sem poder apostar, roubar nem furtar", copJob.badge(), cop.User, suspect.User, formatWait(a.crimeAgo), formatWait(sentence)) + jailFactorText(suspect, suspectJob) + paidFineText(a.fine, cop)
}

func (pa powerAbuse) nextText() string {
	if !pa.presidential {
		return fmt.Sprintf("O próximo abuso de %s só daqui a %s", pa.kind.name, formatWait(pa.kind.every))
	}
	if pa.left == 0 {
		return fmt.Sprintf("Foi o último abuso presidencial de %s do mandato", pa.kind.name)
	}
	return fmt.Sprintf("Sobram %s presidenciais de %s no mandato", countLabel(pa.left, "abuso", "abusos"), pa.kind.name)
}

func (pa powerAbuse) note(text string) string {
	if !pa.used {
		return ""
	}
	return "\n😈 " + text + ". " + pa.nextText()
}

func jailFactorText(suspect types.JID, suspectJob profession) string {
	return professionJailText(suspect, suspectJob) + suspectJob.decrees.note(sentenceDecree)
}

func professionJailText(suspect types.JID, suspectJob profession) string {
	if suspectJob.jailFactor > 0 {
		return fmt.Sprintf("\n⛓️ Como @%s %s, pegou %d%% a mais", suspect.User, suspectJob.title, suspectJob.jailFactor)
	}
	if suspectJob.jailFactor < 0 {
		return fmt.Sprintf("\n🗝️ Como @%s %s, pegou %d%% a menos", suspect.User, suspectJob.title, -suspectJob.jailFactor)
	}
	return ""
}

func arrestingProfessions() string {
	var names []string
	for _, p := range professions {
		if p.jailTime > 0 {
			names = append(names, p.emoji+" "+p.name)
		}
	}
	return joinWithAnd(names)
}

func (b *bot) holdIfJailed(ctx context.Context, evt *events.Message) bool {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	left, err := b.wallets.sentenceLeft(ctx, chat, b.account(ctx, person), time.Now())
	if err != nil {
		b.client.Log.Errorf("checar se tá preso: %v", err)
		return false
	}
	if left == 0 {
		return false
	}
	text := fmt.Sprintf("🔒 @%s tá na cadeia e só sai em %s. Até lá nada de apostar, roubar, furtar, sequestrar, matar nem assaltar, a não ser que consiga !fugir, alguém te !libertar ou você pague a !fianca", person.User, formatWait(left))
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{person}); err != nil {
		b.client.Log.Errorf("avisar que tá preso: %v", err)
	}
	return true
}

func (s *walletStore) arrest(ctx context.Context, chat types.JID, cop, suspect account, cd cooldown, abuse abuser, jail time.Duration, chance float64, collecting, copKeepsFine bool, now time.Time) (a arrest, err error) {
	err = s.inTx(ctx, func(tx *sql.Tx) error {
		if err := openWallet(ctx, tx, chat, suspect); err != nil {
			return err
		}
		var until, crime int64
		err := tx.QueryRowContext(ctx, "SELECT preso_ate, crime_em FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), suspect.key).Scan(&until, &crime)
		if err != nil {
			return err
		}
		if held := time.Unix(until, 0).Sub(now); held > 0 {
			a.held = held
			return nil
		}
		if a.crimeAgo = now.Sub(time.Unix(crime, 0)); !collecting && (crime == 0 || a.crimeAgo > crimeWindow) {
			a.clean = true
			return nil
		}
		if a.wait, a.abuse, err = stampOrAbuse(ctx, tx, chat, cop, cd, abuse, now); err != nil || a.wait > 0 {
			return err
		}
		if a.dismissed = rand.Float64() >= chance; a.dismissed {
			return nil
		}
		query := "UPDATE carteiras SET preso_ate = ?, crime_em = 0 WHERE chat = ? AND pessoa = ?"
		if collecting {
			query = "UPDATE carteiras SET preso_ate = ? WHERE chat = ? AND pessoa = ?"
		}
		_, err = tx.ExecContext(ctx, query, now.Add(jail).Unix(), chat.String(), suspect.key)
		if err != nil {
			return err
		}
		if a.fine, err = lockUp(ctx, tx, chat, suspect, now); err != nil {
			return err
		}
		if copKeepsFine && a.fine > 0 {
			_, err = tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", a.fine, chat.String(), cop.key)
			if err != nil {
				return err
			}
		}
		if a.seized, err = seizeContraband(ctx, tx, chat, suspect); err != nil {
			return err
		}
		a.freed, err = freeHostage(ctx, tx, chat, suspect, now)
		return err
	})
	return a, err
}

func lockUp(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) (int64, error) {
	if err := easeEscape(ctx, tx, chat, acc, now); err != nil {
		return 0, err
	}
	return burnBalance(ctx, tx, chat, acc, fineMinPercent, fineMaxPercent, 1)
}

func fineText(fine int64) string {
	if fine == 0 {
		return ""
	}
	return fmt.Sprintf("\n💸 E ainda pagou %s de multa", coins(fine))
}

func paidFineText(fine int64, payee types.JID) string {
	if fine == 0 {
		return ""
	}
	return fmt.Sprintf("\n💸 E ainda pagou %s de multa pra @%s", coins(fine), payee.User)
}

func stampOrAbuse(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, cd cooldown, abuse abuser, now time.Time) (time.Duration, powerAbuse, error) {
	wait, err := stampCooldown(ctx, tx, chat, acc, cd, now)
	if err != nil || wait == 0 || abuse == nil {
		return wait, powerAbuse{}, err
	}
	pa, err := abuse(ctx, tx, chat, acc, now)
	if err != nil || !pa.used {
		return wait, powerAbuse{}, err
	}
	return 0, pa, nil
}

func markCrime(ctx context.Context, tx *sql.Tx, chat types.JID, acc account, now time.Time) error {
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET crime_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.key)
	return err
}

func (s *walletStore) sentenceLeft(ctx context.Context, chat types.JID, acc account, now time.Time) (time.Duration, error) {
	return s.leftUntil(ctx, chat, acc, "preso_ate", now)
}

func (s *walletStore) deathLeft(ctx context.Context, chat types.JID, acc account, now time.Time) (time.Duration, error) {
	return s.leftUntil(ctx, chat, acc, "morto_ate", now)
}

func (s *walletStore) leftUntil(ctx context.Context, chat types.JID, acc account, column string, now time.Time) (time.Duration, error) {
	var until int64
	err := s.db.QueryRowContext(ctx, "SELECT "+column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.key).Scan(&until)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return max(time.Unix(until, 0).Sub(now), 0), nil
}
