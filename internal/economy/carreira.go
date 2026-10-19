package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	careerStep  = 10
	careerTop   = 10
	careerFirst = 3 * time.Hour
)

type career struct {
	job    string
	worked time.Duration
}

func careerNeeds(level int) time.Duration {
	return careerFirst * time.Duration(level*level)
}

func (c career) level() int {
	level := 0
	for level < careerTop && c.worked >= careerNeeds(level+1) {
		level++
	}
	return level
}

func (c career) bonus() int64 {
	return int64(c.level() * careerStep)
}

func (p Profession) seasoned(careers map[string]time.Duration) Profession {
	if p.divine {
		return p
	}
	p.career = career{job: p.name, worked: careers[p.name]}
	bonus := p.career.bonus()
	p.min, p.max = p.min*(100+bonus)/100, p.max*(100+bonus)/100
	return p
}

func (c career) perk() string {
	level := c.level()
	if level == careerTop {
		return fmt.Sprintf("🎓 Carreira de %s no nível máximo (%d): +%d%% no !trabalhar (já conta acima)", c.job, careerTop, c.bonus())
	}
	left := core.FormatWait(careerNeeds(level+1) - c.worked)
	if level == 0 {
		return fmt.Sprintf("🎓 Carreira de %s: falta %s de trampo pro nível 1, que paga +%d%% no !trabalhar", c.job, left, careerStep)
	}
	return fmt.Sprintf("🎓 Carreira de %s no nível %d: +%d%% no !trabalhar (já conta acima). Falta %s de trampo pro nível %d", c.job, level, c.bonus(), left, level+1)
}

func (c career) promotion(after career) string {
	if after.level() <= c.level() {
		return ""
	}
	return fmt.Sprintf("\n🎓 Subiu pro nível %d na carreira de %s: o !trabalhar agora paga +%d%%", after.level(), c.job, after.bonus())
}

func careerHelp() string {
	return fmt.Sprintf("Quanto mais tempo trabalha na mesma profissão, mais sobe na carreira dela: cada nível paga +%d%% no trampo, até o nível %d (o primeiro sai com %s de trampo). O que é da profissão também conta, só que bem pouco (%s por vez): a extração e a horta de quem é do ramo, e os crimes e poderes que vêm com ela, como o !prender da polícia, o !reviver do necromante e o !postar de quem é da arte. Trocar de profissão começa outra carreira do zero, mas a antiga fica guardada se você voltar", careerStep, careerTop, core.FormatWait(careerNeeds(1)), core.FormatWait(careerDrill))
}

func (s *walletStore) careers(ctx context.Context, chat types.JID, acc core.Account) (map[string]time.Duration, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT profissao, minutos FROM carreiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	worked := map[string]time.Duration{}
	for rows.Next() {
		var (
			job     string
			minutes int64
		)
		if err := rows.Scan(&job, &minutes); err != nil {
			return nil, err
		}
		worked[job] = time.Duration(minutes) * time.Minute
	}
	return worked, rows.Err()
}

func (s *walletStore) build(ctx context.Context, chat types.JID, acc core.Account, c career, worked time.Duration) (career, error) {
	var minutes int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO carreiras (chat, pessoa, profissao, minutos) VALUES (?, ?, ?, ?)
		 ON CONFLICT (chat, pessoa, profissao) DO UPDATE SET minutos = minutos + excluded.minutos
		 RETURNING minutos`,
		chat.String(), acc.Key, c.job, int64(worked/time.Minute),
	).Scan(&minutes)
	return career{job: c.job, worked: time.Duration(minutes) * time.Minute}, err
}

func (b *bot) promote(ctx context.Context, chat types.JID, acc core.Account, job Profession, worked time.Duration) (string, error) {
	if job.divine {
		return "", nil
	}
	after, err := b.Wallets.build(ctx, chat, acc, job.career, worked)
	if err != nil {
		return "", fmt.Errorf("subir na carreira: %w", err)
	}
	return job.career.promotion(after), nil
}

func (c career) tag() string {
	if c.level() == 0 {
		return ""
	}
	return fmt.Sprintf(" (🎓 nível %d na carreira, +%d%% no !trabalhar)", c.level(), c.bonus())
}

const (
	careerDrill = time.Minute
	drillSlack  = 2 * time.Second
)

func (p Profession) crafts(command string) bool {
	return p.craft.activity == command || p.gatherFaster > 0
}

func (p Profession) farms() bool {
	return p.farmBeds > 0 || p.farmFaster > 0 || p.farmGrade > 0
}

func (b *bot) practice(ctx context.Context, chat types.JID, acc core.Account, job Profession, fits func(Profession) bool, drills int) {
	if job.divine || drills <= 0 {
		return
	}
	base, _ := professionNamed(job.career.job)
	if !fits(base) {
		return
	}
	if _, err := b.Wallets.build(ctx, chat, acc, career{job: base.name}, time.Duration(drills)*careerDrill); err != nil {
		b.Client.Log.Errorf("somar carreira de %s: %v", acc.JID, err)
	}
}

func (b *bot) drillCareer(ctx context.Context, evt *events.Message, cmd core.Command) {
	robs := func(p Profession) bool { return p.robEvery > 0 || p.robBonus > 0 }
	famous := func(p Profession) bool { return p.reach > 0 || p.area == areaArte }
	drills := map[string]struct {
		column string
		fits   func(Profession) bool
	}{
		"roubar":     {"roubo_em", robs},
		"quadrilha":  {"roubo_em", robs},
		"furtar":     {"furto_em", func(p Profession) bool { return p.theftEvery > 0 || p.theftBonus > 0 }},
		"sequestrar": {"sequestro_em", func(p Profession) bool { return p.kidnapEvery > 0 || p.kidnapBonus > 0 }},
		"matar":      {"assassinato_em", func(p Profession) bool { return p.killChance > 0 }},
		"prender":    {"prisao_em", func(p Profession) bool { return p.jailTime > 0 }},
		"libertar":   {"libertacao_em", func(p Profession) bool { return p.releaseChance > 0 }},
		"reviver":    {"ressurreicao_em", func(p Profession) bool { return p.reviveChance > 0 }},
		"denunciar":  {"denuncia_em", func(p Profession) bool { return p.reportBonus > 0 }},
		"assalto":    {"assalto_em", func(p Profession) bool { return p.heistBonus > 0 }},
		"processar":  {"processo_em", func(p Profession) bool { return p.trialEdge > 0 || p.trialBonus > 0 }},
		"abencoar":   {"encanto_em", func(p Profession) bool { return len(p.faith.blessings) > 0 }},
		"amaldicoar": {"encanto_em", func(p Profession) bool { return len(p.faith.curses) > 0 }},
		"postar":     {"postagem_em", famous},
		"publi":      {"publi_em", famous},
	}
	drill, ok := drills[cmd.Name]
	if !ok || cmd.Category.Mode != core.ModeEconomy {
		return
	}
	chat := evt.Info.Chat.ToNonAD()
	acc := b.Account(ctx, evt.Info.Sender)
	job, done, err := b.Wallets.drilled(ctx, chat, acc, drill.column, evt.Info.Timestamp.Add(-drillSlack))
	if err != nil {
		b.Client.Log.Errorf("ler carreira de %s depois do !%s: %v", acc.JID, cmd.Name, err)
		return
	}
	if !done {
		return
	}
	b.practice(ctx, chat, acc, job, drill.fits, 1)
}

func (s *walletStore) drilled(ctx context.Context, chat types.JID, acc core.Account, column string, since time.Time) (Profession, bool, error) {
	var (
		name  string
		stamp int64
	)
	err := s.db.QueryRowContext(ctx, "SELECT profissao, "+column+" FROM carteiras WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key).Scan(&name, &stamp)
	if errors.Is(err, sql.ErrNoRows) {
		return Profession{}, false, nil
	}
	job, _ := professionNamed(name)
	job.career = career{job: job.name}
	return job, err == nil && stamp >= since.Unix(), err
}
