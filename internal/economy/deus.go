package economy

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

var god = Profession{
	name: "deus", emoji: "⚡", title: "é deus", aliases: []string{"deusa"}, min: 400, max: 800,
	every: 12 * time.Minute, divine: true,
	guard: 1, robBonus: 1, theftBonus: 1, kidnapBonus: 1, breakoutBonus: 1, escapeBonus: 1, heistBonus: 1,
	killChance: 1, reviveChance: 1, releaseChance: 1, jailTime: baseJail, jailFactor: -100,
	reportBonus: 1, trialEdge: 3, voteWeight: 3,
	tasks: []string{
		"criou um planeta novo só pra passar o tempo", "transformou água em vinho no churrasco da firma",
		"atendeu as orações do grupo todo de uma vez", "mandou um raio no vizinho que tava com som alto",
		"descansou no sétimo dia e cobrou hora extra", "separou o mar pra não pegar trânsito",
	},
}

const godHintText = "\n\n_⚡ Como dono do bot, você pode virar deus quando quiser: !profissao deus. Pra sair, é só escolher outra profissão_"

func (p Profession) pace(cd cooldown) cooldown {
	if p.divine {
		cd.every = 0
		return cd
	}
	cd.every = p.hasted(cd.every)
	return cd
}

func (p Profession) hasted(every time.Duration) time.Duration {
	return every * time.Duration(max(100+p.haste, 0)) / 100
}

func (b *bot) summonsGod(ctx context.Context, evt *events.Message, query string) bool {
	return slices.Contains(god.matchKeys(), normalizeItemName(query)) && b.IsOwner(ctx, evt)
}

func (b *bot) godHint(ctx context.Context, evt *events.Message) string {
	if !b.IsOwner(ctx, evt) {
		return ""
	}
	return godHintText
}

func divinePerks() []string {
	return []string{
		"👑 Só o dono do bot vira deus, e entra e sai da profissão quando quiser, sem esperar",
		"🧟 !reviver, 🔓 !libertar e 🔪 !matar sempre dão certo e não têm espera",
		fmt.Sprintf("🚔 Pode !prender quem cometeu crime nas últimas %s, por %s, sem espera", core.FormatWait(crimeWindow), core.FormatWait(baseJail)),
		fmt.Sprintf("🦹 !roubar, !furtar e !sequestrar com %d%% de chance e sem espera", percent(clampChance(1))),
		fmt.Sprintf("🛡️ Quase intocável: quem tenta !roubar, !furtar, !sequestrar ou !matar você tem só %d%% de chance, e se for preso sai na hora", percent(clampChance(0))),
		fmt.Sprintf("⚖️ No !processar, conta como %s a mais e não tem espera", core.CountLabel(god.trialEdge, "voto", "votos")),
		fmt.Sprintf("🗳️ Seu voto conta como %d no !votar, no !impeachment e no júri do !processar", god.voteWeight),
		fmt.Sprintf("🌧️ !milagre <valor> faz chover moeda nova pra todo mundo do grupo, até %s pra cada um", Coins(miracleMax)),
		"🔥 !castigo <valor> faz sumir moeda de todo mundo do grupo (ou só de quem você marcar), em número, porcentagem, metade ou tudo",
		fmt.Sprintf("✨ !criar <item> faz aparecer qualquer item na sua mochila, até %d de uma vez", creationMax),
		"🙏 !abencoar e 🕯️ !amaldicoar com qualquer bênção ou maldição, sem errar e sem espera",
	}
}
