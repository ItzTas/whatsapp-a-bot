package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	missionsPerDay   = 3
	missionBonus     = 500
	missionPrizeItem = "caixa-misteriosa"

	workGoal   = "trabalho"
	gatherGoal = "extracao"
	betGoal    = "aposta"
	crimeGoal  = "crime"
	saleGoal   = "venda"
	shopGoal   = "compra"
)

type missionGoal struct {
	id              string
	low, high, step int64
	pay             int64
	task            func(target int64) string
}

var missionGoals = []missionGoal{
	{id: workGoal, low: 3, high: 5, step: 1, pay: 80, task: func(n int64) string { return "Trabalhar " + timesLabel(n) + " no !trabalhar" }},
	{id: gatherGoal, low: 3, high: 6, step: 1, pay: 60, task: func(n int64) string { return "Usar o !pescar, !minerar, !vasculhar ou !catar " + timesLabel(n) }},
	{id: betGoal, low: 200, high: 1_000, step: 100, pay: 35, task: func(n int64) string { return "Ganhar " + Coins(n) + " em apostas no !apostar ou no !cassino" }},
	{id: crimeGoal, low: 1, high: 2, step: 1, pay: 250, task: func(n int64) string { return "Se dar bem no !roubar ou no !furtar " + timesLabel(n) }},
	{id: saleGoal, low: 300, high: 1_500, step: 100, pay: 25, task: func(n int64) string { return "Faturar " + Coins(n) + " vendendo pro mercado no !vender" }},
	{id: shopGoal, low: 300, high: 1_500, step: 100, pay: 25, task: func(n int64) string { return "Gastar " + Coins(n) + " comprando na !loja" }},
}

func timesLabel(n int64) string {
	return core.CountLabel(int(n), "vez", "vezes")
}

type mission struct {
	goal   missionGoal
	target int64
}

func (m mission) reward() int64 {
	return m.target / m.goal.step * m.goal.pay
}

func (m mission) line(got int64) string {
	task := m.goal.task(m.target)
	if got >= m.target {
		return fmt.Sprintf("✅ ~%s~ · ganhou %s", task, Coins(m.reward()))
	}
	return fmt.Sprintf("⬜ %s (%s/%s) · %s", task, core.Thousands(got), core.Thousands(m.target), Coins(m.reward()))
}

func todaysMissions(chat types.JID, acc core.Account, now time.Time) []mission {
	rng := dailyRand(chat, now, "missoes "+acc.Key)
	goals := slices.Clone(missionGoals)
	rng.Shuffle(len(goals), func(i, j int) { goals[i], goals[j] = goals[j], goals[i] })
	missions := make([]mission, missionsPerDay)
	for i, g := range goals[:missionsPerDay] {
		missions[i] = mission{goal: g, target: g.low + rng.Int64N((g.high-g.low)/g.step+1)*g.step}
	}
	return missions
}

func missionDay(now time.Time) string {
	return now.In(core.ReminderZone).Format(time.DateOnly)
}

func missionPrize() item {
	prize, _ := itemByID(missionPrizeItem)
	return prize
}

func (b *bot) missoes(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	progress, err := b.Wallets.missionProgress(ctx, chat, acc, missionDay(now))
	if err != nil {
		return fmt.Errorf("ler missões: %w", err)
	}
	lines := []string{fmt.Sprintf("🎯 *Missões de hoje de @%s*", person.User), ""}
	finished := 0
	for _, m := range todaysMissions(chat, acc, now) {
		got := min(progress[m.goal.id], m.target)
		if got == m.target {
			finished++
		}
		lines = append(lines, m.line(got))
	}
	lines = append(lines, "", missionBonusLine(finished), fmt.Sprintf("_A recompensa cai sozinha quando a missão fecha. Missões novas em %s_", core.FormatWait(nextRestock(now))))
	_, err = b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{person})
	return err
}

func missionBonusLine(finished int) string {
	if finished >= missionsPerDay {
		return fmt.Sprintf("🏅 Fechou as %d e já levou o bônus de hoje", missionsPerDay)
	}
	return fmt.Sprintf("🏅 Fechando as %d: mais %s e %s", missionsPerDay, Coins(missionBonus), missionPrize().label())
}

type missionStep struct {
	acc     core.Account
	mission mission
	day     string
	amount  int64
}

func (s missionStep) doneText(bonus bool) string {
	text := fmt.Sprintf("🎯 @%s fechou a missão do dia _%s_ e ganhou %s", s.acc.JID.User, s.mission.goal.task(s.mission.target), Coins(s.mission.reward()))
	if !bonus {
		return text
	}
	return text + fmt.Sprintf("\n🏅 E fechou as %d de hoje! De bônus: mais %s e %s", missionsPerDay, Coins(missionBonus), missionPrize().label())
}

type missionLog struct {
	mu    sync.Mutex
	steps map[types.JID][]missionStep
}

func (l *missionLog) add(chat types.JID, s missionStep) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.steps == nil {
		l.steps = map[types.JID][]missionStep{}
	}
	l.steps[chat] = append(l.steps[chat], s)
}

func (l *missionLog) take(chat types.JID) []missionStep {
	l.mu.Lock()
	defer l.mu.Unlock()
	steps := l.steps[chat]
	delete(l.steps, chat)
	return steps
}

func (b *bot) advance(chat types.JID, acc core.Account, goal string, amount int64) {
	if amount <= 0 {
		return
	}
	now := time.Now()
	missions := todaysMissions(chat, acc, now)
	i := slices.IndexFunc(missions, func(m mission) bool { return m.goal.id == goal })
	if i < 0 {
		return
	}
	b.missionSteps.add(chat, missionStep{acc: acc, mission: missions[i], day: missionDay(now), amount: amount})
}

func (b *bot) settleMissions(ctx context.Context, chat types.JID) {
	for _, s := range b.missionSteps.take(chat) {
		o, err := b.Wallets.advanceMission(ctx, chat, s)
		if err != nil {
			b.Client.Log.Errorf("avançar a missão %s de %s: %v", s.mission.goal.id, s.acc.JID, err)
			continue
		}
		if !o.done {
			continue
		}
		if _, err := b.SendMentions(ctx, chat, s.doneText(o.bonus), []types.JID{s.acc.JID}); err != nil {
			b.Client.Log.Errorf("avisar a missão %s de %s: %v", s.mission.goal.id, s.acc.JID, err)
		}
	}
}

type missionOutcome struct {
	done  bool
	bonus bool
}

func (s *walletStore) missionProgress(ctx context.Context, chat types.JID, acc core.Account, day string) (progress map[string]int64, err error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT missao, progresso FROM missoes WHERE chat = ? AND pessoa = ? AND dia = ?",
		chat.String(), acc.Key, day,
	)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	progress = map[string]int64{}
	for rows.Next() {
		var (
			goal string
			got  int64
		)
		if err := rows.Scan(&goal, &got); err != nil {
			return nil, err
		}
		progress[goal] = got
	}
	return progress, rows.Err()
}

func (s *walletStore) advanceMission(ctx context.Context, chat types.JID, step missionStep) (o missionOutcome, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		var err error
		if o.done, err = stepMission(ctx, tx, chat, step); err != nil || !o.done {
			return err
		}
		if o.bonus, err = missionsFinished(ctx, tx, chat, step); err != nil {
			return err
		}
		return payMission(ctx, tx, chat, step, o.bonus)
	})
	return o, err
}

func stepMission(ctx context.Context, tx *ledgerTx, chat types.JID, step missionStep) (done bool, err error) {
	_, err = tx.ExecContext(ctx, "DELETE FROM missoes WHERE chat = ? AND pessoa = ? AND dia < ?", chat.String(), step.acc.Key, step.day)
	if err != nil {
		return false, err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO missoes (chat, pessoa, dia, missao, meta) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa, dia, missao) DO NOTHING`,
		chat.String(), step.acc.Key, step.day, step.mission.goal.id, step.mission.target,
	)
	if err != nil {
		return false, err
	}
	var got, target int64
	err = tx.QueryRowContext(ctx,
		`UPDATE missoes SET progresso = MIN(progresso + ?, meta)
		 WHERE chat = ? AND pessoa = ? AND dia = ? AND missao = ? AND progresso < meta
		 RETURNING progresso, meta`,
		step.amount, chat.String(), step.acc.Key, step.day, step.mission.goal.id,
	).Scan(&got, &target)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return got >= target, err
}

func missionsFinished(ctx context.Context, tx *ledgerTx, chat types.JID, step missionStep) (bool, error) {
	var finished int
	err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM missoes WHERE chat = ? AND pessoa = ? AND dia = ? AND progresso >= meta",
		chat.String(), step.acc.Key, step.day,
	).Scan(&finished)
	return finished >= missionsPerDay, err
}

func payMission(ctx context.Context, tx *ledgerTx, chat types.JID, step missionStep, bonus bool) error {
	reward := step.mission.reward()
	if bonus {
		reward += missionBonus
	}
	if err := openWallet(ctx, tx, chat, step.acc); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = saldo + ? WHERE chat = ? AND pessoa = ?", reward, chat.String(), step.acc.Key)
	if err != nil || !bonus {
		return err
	}
	return addItems(ctx, tx, chat, step.acc, missionPrizeItem, 1)
}
