package city

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) cidadedorme(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !cidadedorme só funciona em grupo")
	}
	actions := map[string]cityAction{
		"comecar":  (*cityGame).start,
		"sair":     (*cityGame).leave,
		"cancelar": (*cityGame).cancel,
	}
	action, named := actions[core.FoldLetters(core.FirstPlainArg(core.MessageText(evt.Message)))]
	game, active := b.activeCity(chat)
	if named && !active {
		return b.SendText(ctx, chat, "Não tem partida de Cidade Dorme rolando agora. Manda !cidadedorme pra abrir uma sala 🌃")
	}
	if named {
		text, mentions := action(game, evt.Info)
		return b.postCity(ctx, game, text, mentions)
	}
	player := cityPlayer{jid: evt.Info.Sender.ToNonAD(), ids: b.SenderIdentities(ctx, evt)}
	if !active {
		return b.openCity(ctx, chat, player, evt.Info)
	}
	text, mentions := game.join(evt.Info, player)
	return b.postCity(ctx, game, text, mentions)
}

func (b *bot) activeCity(chat types.JID) (*cityGame, bool) {
	value, ok := b.games.Load(chat.String())
	if !ok {
		return nil, false
	}
	game := value.(*cityGame)
	if !game.over() {
		return game, true
	}
	b.games.CompareAndDelete(chat.String(), game)
	return nil, false
}

func (b *bot) openCity(ctx context.Context, chat types.JID, host cityPlayer, info types.MessageInfo) error {
	game := newCityGame(chat, host)
	if value, loaded := b.games.LoadOrStore(chat.String(), game); loaded {
		open := value.(*cityGame)
		text, mentions := open.join(info, host)
		return b.postCity(ctx, open, text, mentions)
	}
	go b.driveCity(ctx, game)
	text, mentions := game.status(fmt.Sprintf("🌃 *Cidade Dorme*\n\n@%s abriu a sala! Quando a partida começar, cada um recebe um papel secreto no meu privado: assassino 🔪, médico 💉, detetive 🔎 ou cidadão 🧑‍🌾\n\nDe noite, o assassino escolhe alguém pra matar, o médico alguém pra salvar e o detetive alguém pra investigar, tudo no privado. De dia, a cidade discute aqui e decide no !votar quem eliminar. A cidade ganha se pegar todos os assassinos, e os assassinos ganham se ficarem em número igual ao resto", host.jid.User))
	return b.postCity(ctx, game, text, mentions)
}

func (b *bot) postCity(ctx context.Context, game *cityGame, text string, mentions []types.JID) error {
	defer b.syncCity(ctx, game)
	if text == "" {
		return nil
	}
	_, err := b.SendMentions(ctx, game.chat, text, mentions)
	return err
}

func (b *bot) tellCity(ctx context.Context, game *cityGame, text string, mentions []types.JID) {
	if _, err := b.SendMentions(ctx, game.chat, text, mentions); err != nil {
		b.Client.Log.Errorf("mandar mensagem do !cidadedorme: %v", err)
	}
}

func (b *bot) driveCity(ctx context.Context, game *cityGame) {
	defer b.Rescue(ctx, game.chat)
	defer b.releaseCity(ctx, game)
	steps := map[cityPhase]func(context.Context, *cityGame) bool{
		cityLobby: b.closeCityLobby,
		cityNight: b.passCityNight,
		cityDay:   b.passCityDay,
	}
	for {
		step, running := steps[game.currentPhase()]
		if !running {
			return
		}
		if game.timedOut() {
			b.finishCity(ctx, game, fmt.Sprintf("⌛ A partida passou de %s e acabou", core.FormatWait(cityTimeout)), nil)
			return
		}
		if !step(ctx, game) {
			return
		}
	}
}

func (b *bot) releaseCity(ctx context.Context, game *cityGame) {
	b.dropCityPrompts(game)
	b.syncCity(ctx, game)
	if game.over() {
		b.games.CompareAndDelete(game.chat.String(), game)
	}
}

func (b *bot) closeCityLobby(ctx context.Context, game *cityGame) bool {
	if !game.wait(ctx) {
		return false
	}
	if !game.deal() {
		b.tellCity(ctx, game, fmt.Sprintf("😴 Não juntou gente suficiente e a partida foi cancelada (precisa de pelo menos %d jogadores)", cityMinPlayers), nil)
		return false
	}
	b.syncCity(ctx, game)
	label := b.PrivateGroupLabel(ctx, game.chat)
	if unreached := b.briefCity(ctx, game, label); len(unreached) > 0 {
		b.finishCity(ctx, game, fmt.Sprintf("📵 Não consegui mandar o papel no privado de %s, então a partida foi cancelada. Todo mundo que joga precisa receber mensagem minha no privado", core.JoinMentions(unreached)), unreached)
		return false
	}
	b.fallNight(ctx, game, label, "🌃 *A partida começou!*\n\n📨 Mandei o papel de cada um no privado. Não conta pra ninguém 🤫")
	return true
}

func (b *bot) briefCity(ctx context.Context, game *cityGame, label string) []types.JID {
	seats := game.seats()
	unreached := make([]types.JID, len(seats))
	var wg sync.WaitGroup
	for _, seat := range seats {
		wg.Go(func() {
			text, mentions := game.briefing(seat, label)
			if !b.SendPrivate(ctx, game.inboxOf(seat), text, mentions) {
				unreached[seat] = game.jidOf(seat)
			}
		})
	}
	wg.Wait()
	return slices.DeleteFunc(unreached, func(jid types.JID) bool { return jid.IsEmpty() })
}

func (b *bot) fallNight(ctx context.Context, game *cityGame, label, header string) {
	text, mentions := game.nightfall(header)
	b.syncCity(ctx, game)
	b.awaitCityNight(ctx, game)
	var names []string
	var wg sync.WaitGroup
	wg.Go(func() { b.tellCity(ctx, game, text, mentions) })
	wg.Go(func() { names = b.cityNames(ctx, game) })
	wg.Wait()
	for _, seat := range game.nightSeats() {
		wg.Go(func() { b.sendCityMenu(ctx, game, seat, label, names) })
	}
	wg.Wait()
	b.syncCity(ctx, game)
}

func (b *bot) cityNames(ctx context.Context, game *cityGame) []string {
	nicknames := b.ChatNicknames(ctx, game.chat)
	players := game.roster()
	names := make([]string, len(players))
	var wg sync.WaitGroup
	for i, player := range players {
		wg.Go(func() { names[i] = b.DisplayName(ctx, player, nicknames) })
	}
	wg.Wait()
	return names
}

func (b *bot) sendCityMenu(ctx context.Context, game *cityGame, seat int, label string, names []string) {
	if b.sendCityPolls(ctx, game, seat, label, names) {
		return
	}
	menu, targets := game.menu(seat, label)
	b.SendPrivate(ctx, game.inboxOf(seat), menu, targets)
}

func (b *bot) sendCityPolls(ctx context.Context, game *cityGame, seat int, label string, names []string) bool {
	prompt, waiting := game.prompts[seat]
	for _, ballot := range game.ballots(seat, label, names) {
		id, sent := b.SendPrivatePoll(ctx, game.inboxOf(seat), ballot.question, ballot.options)
		if !sent {
			return false
		}
		game.keepPoll(seat, cityPoll{ID: id, Options: ballot.options})
		if waiting {
			prompt.WatchPoll(id, ballot.options)
		}
	}
	return true
}

func (b *bot) awaitCityNight(ctx context.Context, game *cityGame) {
	if game.prompts != nil {
		return
	}
	game.prompts = map[int]*core.PrivatePrompt{}
	for _, seat := range game.nightSeats() {
		prompt, free := b.AwaitPrivate(game.inboxOf(seat), b.cityAnswer(game, seat), game.commandOf(seat))
		if !free {
			b.SendPrivate(ctx, game.inboxOf(seat), "⚠️ Você tá com outra resposta pendente no meu privado (tipo um !jokenpo), então nessa noite do Cidade Dorme não dá pra agir. Termina o outro jogo que na próxima noite funciona", nil)
			continue
		}
		for _, poll := range game.pollsOf(seat) {
			prompt.WatchPoll(poll.ID, poll.Options)
		}
		game.prompts[seat] = prompt
	}
}

func (b *bot) dropCityPrompts(game *cityGame) {
	for _, prompt := range game.prompts {
		b.DropPrivate(prompt)
	}
	game.prompts = nil
}

func (b *bot) passCityNight(ctx context.Context, game *cityGame) bool {
	b.awaitCityNight(ctx, game)
	running := game.wait(ctx)
	b.dropCityPrompts(game)
	if !running {
		return false
	}
	dawn := game.dawn()
	b.syncCity(ctx, game)
	if dawn.probe != cityNobody {
		report, mentions := game.probeReport(dawn)
		b.SendPrivate(ctx, game.inboxOf(dawn.sleuth), report, mentions)
	}
	text, mentions := game.dawnText(dawn)
	if ending, over := game.ending(); over {
		b.finishCity(ctx, game, text+"\n\n"+ending, mentions)
		return false
	}
	text, mentions = game.daybreak(text)
	b.syncCity(ctx, game)
	b.tellCity(ctx, game, text, mentions)
	return true
}

func (b *bot) passCityDay(ctx context.Context, game *cityGame) bool {
	if !game.wait(ctx) {
		return false
	}
	text, mentions := game.verdict()
	b.syncCity(ctx, game)
	if ending, over := game.ending(); over {
		b.finishCity(ctx, game, text+"\n\n"+ending, mentions)
		return false
	}
	b.fallNight(ctx, game, b.PrivateGroupLabel(ctx, game.chat), text)
	return true
}

func (b *bot) finishCity(ctx context.Context, game *cityGame, header string, mentions []types.JID) {
	text, players, closed := game.close(header)
	if !closed {
		return
	}
	b.syncCity(ctx, game)
	b.tellCity(ctx, game, text, append(mentions, players...))
}

func (b *bot) haltCity(ctx context.Context, chat types.JID) {
	game, active := b.activeCity(chat)
	if !active {
		return
	}
	b.finishCity(ctx, game, "❌ O modo cidadedorme foi desligado, então a partida acabou", nil)
}

func (b *bot) cityAnswer(game *cityGame, seat int) core.PrivateAnswer {
	return func(ctx context.Context, evt *events.Message, text string) (bool, error) {
		target, ok := cityNumber(text)
		if !ok {
			return false, b.SendText(ctx, evt.Info.Chat, "Manda só o número da lista 😉\n\n"+game.hint(seat))
		}
		pick := game.choose(seat, target-1)
		if pick.taken {
			b.syncCity(ctx, game)
		}
		for _, ally := range pick.allies {
			b.SendPrivate(ctx, ally.ids, pick.tip, append([]types.JID{game.jidOf(seat)}, pick.mentions...))
		}
		_, err := b.SendRawMentions(ctx, evt.Info.Chat, pick.reply, pick.mentions)
		return false, err
	}
}

func cityNumber(text string) (int, bool) {
	fields := strings.Fields(text)
	if len(fields) > 0 && strings.HasPrefix(fields[0], "!") {
		fields = fields[1:]
	}
	if len(fields) == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(strings.Trim(fields[0], "#."))
	return n, err == nil
}

func (b *bot) nightCommand(ctx context.Context, evt *events.Message) error {
	return b.SendText(ctx, evt.Info.Chat, "🤫 O !matar, o !salvar e o !investigar são no meu privado, de noite. Quem tem um desses papéis recebe a lista lá")
}
