package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	bandeiraDuration   = 45 * time.Second
	bandeiraRegionHint = 15 * time.Second
	bandeiraNameHint   = 30 * time.Second
	bandeiraReward     = 30
)

type bandeiraCountry struct {
	flag    string
	name    string
	aliases []string
}

type bandeiraState struct {
	Country  string        `json:"country"`
	Progress guessProgress `json:"progress"`
}

type bandeiraRegion struct {
	where     string
	countries []bandeiraCountry
}

var bandeiraRegions = []bandeiraRegion{
	{"na América do Sul", []bandeiraCountry{
		{"🇧🇷", "Brasil", nil},
		{"🇦🇷", "Argentina", nil},
		{"🇺🇾", "Uruguai", nil},
		{"🇵🇾", "Paraguai", nil},
		{"🇨🇱", "Chile", nil},
		{"🇧🇴", "Bolívia", nil},
		{"🇵🇪", "Peru", nil},
		{"🇪🇨", "Equador", nil},
		{"🇨🇴", "Colômbia", nil},
		{"🇻🇪", "Venezuela", nil},
		{"🇬🇾", "Guiana", nil},
		{"🇸🇷", "Suriname", nil},
	}},
	{"na América do Norte", []bandeiraCountry{
		{"🇺🇸", "Estados Unidos", []string{"eua", "usa", "estados unidos da america"}},
		{"🇨🇦", "Canadá", nil},
		{"🇲🇽", "México", nil},
	}},
	{"na América Central ou no Caribe", []bandeiraCountry{
		{"🇨🇺", "Cuba", nil},
		{"🇯🇲", "Jamaica", nil},
		{"🇭🇹", "Haiti", nil},
		{"🇩🇴", "República Dominicana", []string{"dominicana"}},
		{"🇵🇷", "Porto Rico", nil},
		{"🇵🇦", "Panamá", nil},
		{"🇨🇷", "Costa Rica", nil},
		{"🇬🇹", "Guatemala", nil},
		{"🇭🇳", "Honduras", nil},
		{"🇳🇮", "Nicarágua", nil},
		{"🇸🇻", "El Salvador", []string{"salvador"}},
		{"🇧🇸", "Bahamas", nil},
	}},
	{"na Europa", []bandeiraCountry{
		{"🇵🇹", "Portugal", nil},
		{"🇪🇸", "Espanha", nil},
		{"🇫🇷", "França", nil},
		{"🇮🇹", "Itália", nil},
		{"🇩🇪", "Alemanha", nil},
		{"🇬🇧", "Reino Unido", []string{"inglaterra", "gra bretanha", "uk"}},
		{"🇮🇪", "Irlanda", nil},
		{"🇳🇱", "Holanda", []string{"paises baixos"}},
		{"🇧🇪", "Bélgica", nil},
		{"🇨🇭", "Suíça", nil},
		{"🇦🇹", "Áustria", nil},
		{"🇵🇱", "Polônia", nil},
		{"🇨🇿", "República Tcheca", []string{"tchequia", "republica checa", "chequia"}},
		{"🇸🇰", "Eslováquia", nil},
		{"🇭🇺", "Hungria", nil},
		{"🇷🇴", "Romênia", nil},
		{"🇧🇬", "Bulgária", nil},
		{"🇬🇷", "Grécia", nil},
		{"🇭🇷", "Croácia", nil},
		{"🇷🇸", "Sérvia", nil},
		{"🇸🇮", "Eslovênia", nil},
		{"🇧🇦", "Bósnia", []string{"bosnia e herzegovina", "bosnia herzegovina"}},
		{"🇦🇱", "Albânia", nil},
		{"🇺🇦", "Ucrânia", nil},
		{"🇷🇺", "Rússia", nil},
		{"🇧🇾", "Belarus", []string{"bielorrussia"}},
		{"🇸🇪", "Suécia", nil},
		{"🇳🇴", "Noruega", nil},
		{"🇩🇰", "Dinamarca", nil},
		{"🇫🇮", "Finlândia", nil},
		{"🇮🇸", "Islândia", nil},
		{"🇪🇪", "Estônia", nil},
		{"🇱🇻", "Letônia", nil},
		{"🇱🇹", "Lituânia", nil},
		{"🇲🇹", "Malta", nil},
		{"🇱🇺", "Luxemburgo", nil},
	}},
	{"no Oriente Médio", []bandeiraCountry{
		{"🇹🇷", "Turquia", nil},
		{"🇸🇦", "Arábia Saudita", []string{"arabia"}},
		{"🇦🇪", "Emirados Árabes Unidos", []string{"emirados", "emirados arabes", "eau", "dubai"}},
		{"🇶🇦", "Catar", []string{"qatar"}},
		{"🇮🇱", "Israel", nil},
		{"🇵🇸", "Palestina", nil},
		{"🇱🇧", "Líbano", nil},
		{"🇸🇾", "Síria", nil},
		{"🇯🇴", "Jordânia", nil},
		{"🇮🇶", "Iraque", nil},
		{"🇮🇷", "Irã", []string{"ira", "iran"}},
		{"🇰🇼", "Kuwait", nil},
		{"🇴🇲", "Omã", nil},
		{"🇾🇪", "Iêmen", nil},
	}},
	{"na Ásia", []bandeiraCountry{
		{"🇯🇵", "Japão", nil},
		{"🇨🇳", "China", nil},
		{"🇰🇷", "Coreia do Sul", []string{"korea do sul"}},
		{"🇰🇵", "Coreia do Norte", []string{"korea do norte"}},
		{"🇮🇳", "Índia", nil},
		{"🇵🇰", "Paquistão", nil},
		{"🇧🇩", "Bangladesh", nil},
		{"🇹🇭", "Tailândia", nil},
		{"🇻🇳", "Vietnã", []string{"vietna", "vietnam"}},
		{"🇵🇭", "Filipinas", nil},
		{"🇮🇩", "Indonésia", nil},
		{"🇲🇾", "Malásia", nil},
		{"🇸🇬", "Singapura", []string{"cingapura"}},
		{"🇰🇭", "Camboja", nil},
		{"🇳🇵", "Nepal", nil},
		{"🇱🇰", "Sri Lanka", nil},
		{"🇲🇳", "Mongólia", nil},
		{"🇰🇿", "Cazaquistão", nil},
		{"🇦🇫", "Afeganistão", nil},
		{"🇹🇼", "Taiwan", nil},
	}},
	{"na África", []bandeiraCountry{
		{"🇿🇦", "África do Sul", nil},
		{"🇳🇬", "Nigéria", nil},
		{"🇪🇬", "Egito", nil},
		{"🇲🇦", "Marrocos", nil},
		{"🇩🇿", "Argélia", nil},
		{"🇹🇳", "Tunísia", nil},
		{"🇱🇾", "Líbia", nil},
		{"🇸🇳", "Senegal", nil},
		{"🇬🇭", "Gana", nil},
		{"🇨🇮", "Costa do Marfim", nil},
		{"🇨🇲", "Camarões", nil},
		{"🇰🇪", "Quênia", nil},
		{"🇪🇹", "Etiópia", nil},
		{"🇦🇴", "Angola", nil},
		{"🇲🇿", "Moçambique", nil},
		{"🇨🇻", "Cabo Verde", nil},
		{"🇲🇬", "Madagascar", []string{"madagasgar"}},
		{"🇸🇩", "Sudão", nil},
		{"🇺🇬", "Uganda", nil},
		{"🇷🇼", "Ruanda", nil},
		{"🇹🇿", "Tanzânia", nil},
		{"🇿🇼", "Zimbábue", nil},
	}},
	{"na Oceania", []bandeiraCountry{
		{"🇦🇺", "Austrália", nil},
		{"🇳🇿", "Nova Zelândia", nil},
		{"🇫🇯", "Fiji", nil},
		{"🇵🇬", "Papua-Nova Guiné", []string{"papua nova guine", "papua"}},
	}},
}

func (b *bot) bandeira(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if b.guessRunning(chat) {
		return b.replyTo(ctx, evt, guessBusy)
	}
	value, _ := b.bandeiraRecent.LoadOrStore(chat.String(), &recentAnswers{})
	recent := value.(*recentAnswers)
	country, where := pickBandeira(recent.list())
	answers := country.answers()
	round := newGuessRound(func(guess string) bool { return matchesTitle(guess, answers) }, nil)
	b.keepBandeira(ctx, chat.String(), round, country)
	if !b.startGuessRound(chat, round) {
		return b.replyTo(ctx, evt, guessBusy)
	}
	recent.add(country.name)

	intro := fmt.Sprintf("🏳️ *De que país é essa bandeira?*\n\n%s\n\n_Manda o nome do país. Tem dica em %d segundos e a resposta sai em %d_ ⏳",
		country.flag, int(bandeiraRegionHint.Seconds()), int(bandeiraDuration.Seconds()))
	if err := b.sendText(ctx, chat, intro); err != nil {
		b.abandonGuess(chat, round)
		return err
	}
	round.keeper.keep()
	go b.finishBandeira(ctx, chat, round, country, where)
	return nil
}

func (b *bot) keepBandeira(ctx context.Context, key string, round *guessRound, country bandeiraCountry) {
	b.keepRound(ctx, &round.keeper, "bandeira", key, round.started.Add(bandeiraDuration), func() any {
		return bandeiraState{Country: country.name, Progress: round.progress()}
	})
}

func (b *bot) restoreBandeira(ctx context.Context, saved savedGame) error {
	var state bandeiraState
	if err := json.Unmarshal(saved.state, &state); err != nil {
		return fmt.Errorf("ler rodada do !bandeira: %w", err)
	}
	country, where, ok := findBandeira(state.Country)
	if !ok {
		return fmt.Errorf("o país %q não tá mais na lista do !bandeira", state.Country)
	}
	answers := country.answers()
	round := restoreGuessRound(func(guess string) bool { return matchesTitle(guess, answers) }, nil, state.Progress)
	b.keepBandeira(ctx, saved.key, round, country)
	chat, err := b.resumeGuess(saved, round)
	if err != nil {
		return err
	}
	go b.finishBandeira(ctx, chat, round, country, where)
	return nil
}

func (b *bot) finishBandeira(ctx context.Context, chat types.JID, round *guessRound, country bandeiraCountry, where string) {
	hints := []guessHint{
		{after: bandeiraRegionHint, text: fmt.Sprintf("🌎 Dica: %s fica %s", country.flag, where)},
		{after: bandeiraNameHint, text: "💡 Dica: " + titleHint(country.name)},
	}
	if err := b.awaitGuess(ctx, chat, round, bandeiraDuration, hints); err != nil {
		return
	}
	winner, solved, players := round.outcome()
	if !solved {
		if err := b.sendText(ctx, chat, fmt.Sprintf("%s\n\n⏰ Ninguém acertou! Era *%s*", country.flag, country.name)); err != nil {
			b.client.Log.Errorf("revelar !bandeira: %v", err)
		}
		return
	}
	text := fmt.Sprintf("%s\n\n🎉 @%s acertou! Era *%s*", country.flag, winner.User, country.name)
	text = appendLine(text, b.reward(ctx, chat, []types.JID{winner}, players, bandeiraReward))
	if _, err := b.sendMentions(ctx, chat, text, []types.JID{winner}); err != nil {
		b.client.Log.Errorf("revelar !bandeira: %v", err)
	}
}

func findBandeira(name string) (bandeiraCountry, string, bool) {
	for _, region := range bandeiraRegions {
		for _, country := range region.countries {
			if country.name == name {
				return country, region.where, true
			}
		}
	}
	return bandeiraCountry{}, "", false
}

func pickBandeira(avoid []string) (bandeiraCountry, string) {
	var (
		picked      bandeiraCountry
		pickedWhere string
		seen        int
	)
	for _, region := range bandeiraRegions {
		for _, country := range region.countries {
			if slices.Contains(avoid, country.name) {
				continue
			}
			seen++
			if rand.IntN(seen) == 0 {
				picked, pickedWhere = country, region.where
			}
		}
	}
	return picked, pickedWhere
}

func (c bandeiraCountry) answers() []string {
	var answers []string
	for _, name := range append([]string{c.name}, c.aliases...) {
		if title := titleKey(name); title != "" && !slices.Contains(answers, title) {
			answers = append(answers, title)
		}
	}
	return answers
}
