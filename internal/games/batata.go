package games

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	batataKind    = "batata"
	batataMinFuse = 30 * time.Second
	batataMaxFuse = 90 * time.Second
)

type batataGame struct {
	mu        sync.Mutex
	holder    types.JID
	holderIDs []types.JID
	giverIDs  []types.JID
	passes    int
	over      bool
	lit       time.Time
	fuse      time.Duration
	record    func()
}

type batataSave struct {
	Chat      types.JID     `json:"chat"`
	Holder    types.JID     `json:"holder"`
	HolderIDs []types.JID   `json:"holder_ids"`
	GiverIDs  []types.JID   `json:"giver_ids"`
	Passes    int           `json:"passes"`
	Lit       time.Time     `json:"lit"`
	Fuse      time.Duration `json:"fuse"`
}

func (b *bot) batata(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	if chat.Server != types.GroupServer {
		return b.SendText(ctx, chat, "O !batata só funciona em grupo")
	}
	game, active := b.activeBatata(chat)
	if !active {
		return b.startBatata(ctx, evt)
	}
	if game.heldBy(evt.Info) && len(core.MentionedJIDs(evt.Message)) > 0 {
		return b.passBatata(ctx, evt, game)
	}
	return b.showBatata(ctx, chat, game)
}

func (b *bot) answerBatata(ctx context.Context, evt *events.Message) error {
	if len(core.MentionedJIDs(evt.Message)) == 0 {
		return nil
	}
	if name, _ := core.CommandName(core.MessageText(evt.Message)); name == "batata" {
		return nil
	}
	game, active := b.activeBatata(evt.Info.Chat.ToNonAD())
	if !active || !game.heldBy(evt.Info) {
		return nil
	}
	return b.passBatata(ctx, evt, game)
}

func (b *bot) activeBatata(chat types.JID) (*batataGame, bool) {
	value, ok := b.batataGames.Load(chat.String())
	if !ok {
		return nil, false
	}
	return value.(*batataGame), true
}

func (b *bot) startBatata(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	holder, ids := evt.Info.Sender.ToNonAD(), b.SenderIdentities(ctx, evt)
	if mentions := core.MentionedJIDs(evt.Message); len(mentions) > 0 {
		holder = mentions[0].ToNonAD()
		ids = b.Identities(ctx, holder)
	}
	if b.IsSelf(holder) {
		return b.ReplyTo(ctx, evt, "🤖 Comigo não, eu não tenho mão. Marca outra pessoa")
	}
	fuse := batataMinFuse + rand.N(batataMaxFuse-batataMinFuse)
	game := b.trackBatata(ctx, chat, &batataGame{holder: holder, holderIDs: ids, lit: time.Now(), fuse: fuse})
	if value, loaded := b.batataGames.LoadOrStore(chat.String(), game); loaded {
		return b.showBatata(ctx, chat, value.(*batataGame))
	}
	game.save()
	go b.explodeBatata(ctx, chat, game)
	text := fmt.Sprintf("🥔🔥 *Batata quente!*\n\nA batata tá na mão de @%s\n\n_Pra passar, é só mandar qualquer mensagem marcando alguém. O pavio tá aceso e ninguém sabe quando ela explode_ 💣", holder.User)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{holder})
	return err
}

func (b *bot) showBatata(ctx context.Context, chat types.JID, game *batataGame) error {
	holder := game.currentHolder()
	text := fmt.Sprintf("🥔🔥 A batata tá na mão de @%s! Marca alguém pra se livrar dela", holder.User)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{holder})
	return err
}

func (b *bot) passBatata(ctx context.Context, evt *events.Message, game *batataGame) error {
	from := evt.Info.Sender.ToNonAD()
	var refusal string
	for _, mention := range core.MentionedJIDs(evt.Message) {
		target := mention.ToNonAD()
		if b.IsSelf(target) {
			refusal = cmp.Or(refusal, "🤖 Comigo não, eu não tenho mão")
			continue
		}
		why, passed := game.pass(evt.Info, target, b.Identities(ctx, target))
		if passed {
			text := fmt.Sprintf("🥔 @%s jogou a batata pra @%s", from.User, target.User)
			_, err := b.SendMentions(ctx, evt.Info.Chat, text, []types.JID{from, target})
			return err
		}
		refusal = cmp.Or(refusal, why)
	}
	if refusal == "" {
		return nil
	}
	return b.ReplyTo(ctx, evt, refusal)
}

func (b *bot) trackBatata(ctx context.Context, chat types.JID, game *batataGame) *batataGame {
	game.record = func() { b.recordBatata(ctx, chat, game) }
	return game
}

func (b *bot) recordBatata(ctx context.Context, chat types.JID, game *batataGame) {
	if game.over {
		b.DropGame(ctx, batataKind, chat.String())
		return
	}
	b.KeepGame(ctx, batataKind, chat.String(), game.saved(chat), game.lit.Add(game.fuse))
}

func (b *bot) restoreBatata(ctx context.Context, saved core.SavedGame) error {
	var state batataSave
	if err := json.Unmarshal(saved.State, &state); err != nil {
		return fmt.Errorf("ler a !batata: %w", err)
	}
	game := b.trackBatata(ctx, state.Chat, &batataGame{
		holder:    state.Holder,
		holderIDs: state.HolderIDs,
		giverIDs:  state.GiverIDs,
		passes:    state.Passes,
		lit:       state.Lit,
		fuse:      state.Fuse,
	})
	if time.Now().After(game.lit.Add(game.fuse)) {
		return b.coolBatata(ctx, state.Chat, game)
	}
	if _, loaded := b.batataGames.LoadOrStore(saved.Key, game); loaded {
		return nil
	}
	go b.explodeBatata(ctx, state.Chat, game)
	return nil
}

func (b *bot) coolBatata(ctx context.Context, chat types.JID, game *batataGame) error {
	holder, _ := game.explode()
	text := fmt.Sprintf("🥔❄️ O bot reiniciou e a batata esfriou na mão de @%s antes de explodir. Dessa vez ninguém perdeu", holder.User)
	_, err := b.SendMentions(ctx, chat, text, []types.JID{holder})
	return err
}

func (b *bot) explodeBatata(ctx context.Context, chat types.JID, game *batataGame) {
	defer b.batataGames.CompareAndDelete(chat.String(), game)
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Until(game.lit.Add(game.fuse))):
	}
	holder, passes := game.explode()
	endings := []string{
		"🍟 Virou batata frita",
		"🥔 Agora é purê",
		"💀 Descanse em purê",
		"🧑‍🚒 Chamaram os bombeiros, mas já era",
		"🔥 Ficou mais tostado que churrasco de domingo",
		"🤡 Segurou igual segura contatinho: até queimar",
		"📸 Alguém tirou print do momento exato",
		"🧤 Da próxima vez vem de luva",
		"😵 Vai ficar com a mão cheirando a batata pra sempre",
	}
	lines := []string{
		fmt.Sprintf("💥 *BUM!* A batata explodiu na mão de @%s", holder.User),
		core.PickOne(endings),
		"",
		batataSummary(passes, game.fuse),
	}
	if _, err := b.SendMentions(ctx, chat, strings.Join(lines, "\n"), []types.JID{holder}); err != nil {
		b.Client.Log.Errorf("explodir !batata: %v", err)
	}
}

func batataSummary(passes int, fuse time.Duration) string {
	seconds := int(fuse.Seconds())
	if passes == 0 {
		return fmt.Sprintf("🙄 Ficou %d segundos parada na mesma mão, ninguém nem tentou passar", seconds)
	}
	unit := "vezes"
	if passes == 1 {
		unit = "vez"
	}
	return fmt.Sprintf("🔁 Ela passou de mão %d %s em %d segundos", passes, unit, seconds)
}

func (g *batataGame) heldBy(info types.MessageInfo) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return !g.over && g.sentByHolder(info)
}

func (g *batataGame) currentHolder() types.JID {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.holder
}

func (g *batataGame) pass(info types.MessageInfo, target types.JID, targetIDs []types.JID) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.over || !g.sentByHolder(info) {
		return "", false
	}
	if slices.ContainsFunc(targetIDs, func(id types.JID) bool { return core.SentBy(info, id) }) {
		return "🙄 Passar pra você mesmo não vale", false
	}
	if slices.ContainsFunc(targetIDs, func(id types.JID) bool { return slices.Contains(g.giverIDs, id) }) {
		return "↩️ Não pode devolver direto pra quem te passou, marca outra pessoa", false
	}
	g.giverIDs = g.holderIDs
	g.holder, g.holderIDs = target, targetIDs
	g.passes++
	g.record()
	return "", true
}

func (g *batataGame) explode() (types.JID, int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.over = true
	g.record()
	return g.holder, g.passes
}

func (g *batataGame) save() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.record()
}

func (g *batataGame) saved(chat types.JID) batataSave {
	return batataSave{
		Chat:      chat,
		Holder:    g.holder,
		HolderIDs: g.holderIDs,
		GiverIDs:  g.giverIDs,
		Passes:    g.passes,
		Lit:       g.lit,
		Fuse:      g.fuse,
	}
}

func (g *batataGame) sentByHolder(info types.MessageInfo) bool {
	return slices.ContainsFunc(g.holderIDs, func(id types.JID) bool { return core.SentBy(info, id) })
}
