package economy

import (
	"context"
	"fmt"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

const botchedRiskFactor = 2

type crimeRisk struct {
	command string
	emoji   string
	self    hazard
	victim  hazard
}

var (
	robRisk = crimeRisk{command: "!roubar", emoji: "🦹",
		self: hazard{chance: 0.05, harm: 25, wounds: []string{
			"cortou a mão pulando o muro da vítima", "levou uma mordida do cachorro do vizinho", "torceu o tornozelo fugindo pela calçada",
		}},
		victim: hazard{chance: 0.25, harm: 20, wounds: []string{
			"levou uma coronhada do ladrão", "caiu de cara no chão quando puxaram a bolsa", "levou um empurrão e bateu a cabeça no poste",
		}},
	}
	theftRisk = crimeRisk{command: "!furtar", emoji: "🥷",
		self: hazard{chance: 0.03, harm: 15, wounds: []string{
			"prendeu o dedo no zíper da mochila alheia", "tropeçou no meio da multidão e ralou o joelho", "levou uma guarda-chuvada de uma senhora atenta",
		}},
	}
	kidnapRisk = crimeRisk{command: "!sequestrar", emoji: "🚐",
		self: hazard{chance: 0.06, harm: 30, wounds: []string{
			"levou um chute da vítima bem no meio das pernas", "bateu a cabeça na porta da van", "foi mordido no braço na hora de amarrar",
		}},
		victim: hazard{chance: 0.4, harm: 25, wounds: []string{
			"bateu a cabeça no teto da van", "ficou com o pulso machucado de tanto puxar a corda", "levou uns tapas pra parar de gritar",
		}},
	}
	killRisk = crimeRisk{command: "!matar", emoji: "🔪",
		self: hazard{chance: 0.08, harm: 40, wounds: []string{
			"levou uma facada da vítima, que reagiu", "deu um tiro no próprio pé", "caiu da laje fugindo da cena do crime",
		}},
		victim: hazard{chance: 0.5, harm: 35, wounds: []string{
			"levou um tiro de raspão", "levou uma facada no braço, mas correu a tempo", "rolou escada abaixo fugindo do assassino",
		}},
	}
	heistRisk = crimeRisk{command: "!assalto", emoji: "🏦",
		self: hazard{chance: 0.08, harm: 40, wounds: []string{
			"levou um tiro do segurança", "cortou a mão no vidro do caixa", "ficou preso na porta giratória e saiu roxo",
		}},
	}
	gangRisk = crimeRisk{command: "!quadrilha", emoji: "👥",
		self: hazard{chance: 0.04, harm: 20, wounds: []string{
			"levou uma bolsada na cara", "tomou um soco de um vizinho que veio ajudar", "torceu o pé correndo da confusão",
		}},
		victim: hazard{chance: 0.35, harm: 30, wounds: []string{
			"apanhou da quadrilha inteira", "levou uma rasteira e caiu de mau jeito", "ficou com o olho roxo depois do cerco",
		}},
	}
	crimeRisks = []crimeRisk{robRisk, theftRisk, kidnapRisk, killRisk, heistRisk, gangRisk}
)

type scuffle struct {
	criminal injury
	victim   injury
}

func (r crimeRisk) by(p Profession) crimeRisk {
	if p.divine {
		r.self = hazard{}
	}
	return r
}

func (r crimeRisk) on(p Profession) crimeRisk {
	if p.divine {
		r.victim = hazard{}
	}
	return r
}

func (r crimeRisk) wound(success bool) wound {
	h := r.self
	if !success {
		h.chance *= botchedRiskFactor
	}
	return h.roll(0)
}

func (r crimeRisk) clash(ctx context.Context, tx *ledgerTx, chat types.JID, criminal, victim core.Account, success, struck bool, now time.Time) (s scuffle, err error) {
	if s.criminal, err = hurt(ctx, tx, chat, criminal, r.wound(success), now); err != nil || !struck {
		return s, err
	}
	s.victim, err = bruise(ctx, tx, chat, victim, r.victim.roll(0), now)
	return s, err
}

func bruise(ctx context.Context, tx *ledgerTx, chat types.JID, acc core.Account, w wound, now time.Time) (inj injury, err error) {
	if w.damage == 0 {
		return inj, nil
	}
	current, err := readHealth(ctx, tx, chat, acc, now)
	if err != nil || current <= 1 {
		return inj, err
	}
	inj.wound = w
	inj.health = max(current-w.damage, 1)
	inj.damage = current - inj.health
	return inj, setHealth(ctx, tx, chat, acc, inj.health, now)
}

func (s scuffle) text(criminal, victim types.JID) (string, []types.JID) {
	criminalText, freed := s.criminal.text(criminal)
	victimText, _ := s.victim.text(victim)
	return criminalText + victimText, freed
}

func (r crimeRisk) line(job Profession) string {
	h := r.by(job).self
	if h.harm == 0 {
		return fmt.Sprintf("%s %s: não machuca", r.emoji, r.command)
	}
	botched := h
	botched.chance *= botchedRiskFactor
	return fmt.Sprintf("%s %s: %s de chance (%s se der errado), até %d de saúde", r.emoji, r.command, oddsText(h.odds(0)), oddsText(botched.odds(0)), h.harm)
}
