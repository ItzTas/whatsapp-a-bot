package economy

import (
	"context"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/types"
)

var (
	eventPlaces  = []string{"!pescar", "!minerar", "!vasculhar", "!catar", "!trabalhar", "!roubar", "!furtar", "!sequestrar", "!matar", "!assalto", "!quadrilha", "!mercadonegro", "!postar", "!publi", "!apostar", "!cassino", "!blackjack", "!crash", "!corrida"}
	victimPlaces = []string{"!roubar", "!furtar", "!sequestrar", "!matar", "!quadrilha"}
	placeholder  = regexp.MustCompile(`\{[^}]*\}`)
	offCharms    = []string{"dizimo", "pacto", "assombracao"}
	harshCharms  = []string{"pacto", "assombracao"}
)

func TestEventCatalog(t *testing.T) {
	ids := map[string]int{}
	for _, sc := range eventCatalog {
		ids[sc.id]++
	}
	referenced := map[string]bool{}
	for _, sc := range eventCatalog {
		checkScenario(t, sc, ids, referenced)
	}
	for _, sc := range eventCatalog {
		if sc.stage && !referenced[sc.id] {
			t.Errorf("%s: etapa que nenhum desfecho abre (next ou plant)", sc.id)
		}
	}
}

func checkScenario(t *testing.T, sc listedScenario, ids map[string]int, referenced map[string]bool) {
	t.Helper()
	if sc.id == "" || ids[sc.id] > 1 {
		t.Errorf("%q: id vazio ou repetido", sc.id)
	}
	if strings.TrimSpace(sc.title) == "" || len(sc.intros) == 0 {
		t.Errorf("%s: falta título ou abertura", sc.id)
	}
	for _, p := range sc.places {
		if !slices.Contains(eventPlaces, p) {
			t.Errorf("%s: lugar desconhecido %q", sc.id, p)
		}
	}
	for _, a := range sc.areas {
		if a < 0 || a >= len(areas) {
			t.Errorf("%s: área inválida %d", sc.id, a)
		}
	}
	for _, j := range sc.jobs {
		if _, ok := professionNamed(j); !ok {
			t.Errorf("%s: profissão desconhecida %q", sc.id, j)
		}
	}
	victimOK := len(sc.places) > 0 && !slices.ContainsFunc(sc.places, func(p string) bool { return !slices.Contains(victimPlaces, p) })
	texts := slices.Clone(sc.intros)
	free := 0
	labels := map[string]bool{}
	if len(sc.choices) < 2 || len(sc.choices) > 5 {
		t.Errorf("%s: precisa de 2 a 5 opções, tem %d", sc.id, len(sc.choices))
	}
	for _, ch := range sc.choices {
		if ch.needs == nil && ch.uses == "" && ch.price == (cost{}) {
			free++
		}
		if ch.label == "" || labels[ch.label] || utf8.RuneCountInString(ch.label) > 45 {
			t.Errorf("%s: opção vazia, repetida ou longa demais: %q", sc.id, ch.label)
		}
		labels[ch.label] = true
		if ch.uses != "" {
			if _, ok := itemByID(ch.uses); !ok {
				t.Errorf("%s: item gasto desconhecido %q", sc.id, ch.uses)
			}
		}
		if len(ch.endings) == 0 {
			t.Errorf("%s/%s: opção sem desfecho", sc.id, ch.label)
		}
		for _, e := range ch.endings {
			texts = append(texts, e.texts...)
			checkEnding(t, sc, ch.label, e, ids, referenced)
		}
	}
	if len(sc.idle.texts) > 0 || len(sc.idle.effects) > 0 {
		texts = append(texts, sc.idle.texts...)
		checkEnding(t, sc, "idle", sc.idle, ids, referenced)
	}
	if free < 2 {
		t.Errorf("%s: precisa de pelo menos 2 opções sem needs, uses e price (tem %d)", sc.id, free)
	}
	for _, text := range texts {
		checkText(t, sc.id, text, victimOK)
	}
}

func checkEnding(t *testing.T, sc listedScenario, label string, e ending, ids map[string]int, referenced map[string]bool) {
	t.Helper()
	if label != "idle" && e.weight <= 0 {
		t.Errorf("%s/%s: desfecho sem peso", sc.id, label)
	}
	if len(e.texts) == 0 {
		t.Errorf("%s/%s: desfecho sem texto", sc.id, label)
	}
	if e.next != "" {
		referenced[e.next] = true
		if next, ok := scenarioByID(e.next); !ok || !next.stage {
			t.Errorf("%s/%s: next %q não existe ou não é etapa", sc.id, label, e.next)
		}
	}
	for _, fx := range e.effects {
		checkEffect(t, sc, label, fx, ids, referenced)
		checkPlaces(t, sc, label, fx)
	}
}

func checkPlaces(t *testing.T, sc listedScenario, label string, fx effect) {
	t.Helper()
	places := sc.places
	if len(places) == 0 {
		places = eventPlaces
	}
	tooled := func(p string) bool {
		return p == "!trabalhar" || slices.ContainsFunc(activities, func(a activity) bool { return a.command == p && a.tool != "" })
	}
	for _, p := range places {
		var idle bool
		switch fx.(type) {
		case breakEffect:
			idle = !tooled(p)
		case refundEffect:
			idle = !slices.Contains(victimPlaces, p)
		}
		if idle {
			t.Errorf("%s/%s: %T não faz nada em %s", sc.id, label, fx, p)
		}
	}
}

func checkEffect(t *testing.T, sc listedScenario, label string, fx effect, ids map[string]int, referenced map[string]bool) {
	t.Helper()
	switch v := fx.(type) {
	case gainEffect:
		if len(v.ids) == 0 || v.n <= 0 {
			t.Errorf("%s/%s: gain vazio", sc.id, label)
		}
		for _, id := range v.ids {
			checkPrize(t, sc, label, id)
		}
	case forfeitEffect:
		if _, ok := itemByID(v.id); !ok {
			t.Errorf("%s/%s: forfeit de item desconhecido %q", sc.id, label, v.id)
		}
	case buzzEffect:
		if it, ok := itemByID(v.id); !ok || it.high == nil {
			t.Errorf("%s/%s: buzz em algo que não é droga %q", sc.id, label, v.id)
		}
	case spellEffect:
		if _, ok := charmByID(v.id); !ok || slices.Contains(offCharms, v.id) {
			t.Errorf("%s/%s: encanto inválido %q", sc.id, label, v.id)
		}
	case grudgeEffect:
		if c, ok := charmByID(v.id); !ok || !c.curse {
			t.Errorf("%s/%s: praga da vítima precisa ser maldição: %q", sc.id, label, v.id)
		}
		if slices.Contains(harshCharms, v.id) && sc.rarity == commonEvent {
			t.Errorf("%s/%s: %s só em evento raro ou lendário", sc.id, label, v.id)
		}
	case plantEffect:
		referenced[v.id] = true
		if ids[v.id] == 0 || v.after <= 0 {
			t.Errorf("%s/%s: plant de evento desconhecido %q ou sem prazo", sc.id, label, v.id)
		}
	case earnEffect, loseEffect, injureEffect, refundEffect, fansEffect, unfollowEffect:
		checkRange(t, sc, label, fx)
	}
}

func checkRange(t *testing.T, sc listedScenario, label string, fx effect) {
	t.Helper()
	bounds := map[string][2]int64{}
	switch v := fx.(type) {
	case earnEffect:
		bounds["earn"] = [2]int64{v.lo, v.hi}
	case loseEffect:
		bounds["lose"] = [2]int64{v.lo, v.hi}
		if v.hi > 30 {
			t.Errorf("%s/%s: lose acima de 30%% do saldo", sc.id, label)
		}
	case injureEffect:
		bounds["injure"] = [2]int64{v.lo, v.hi}
	case fansEffect:
		bounds["fans"] = [2]int64{v.lo, v.hi}
	case unfollowEffect:
		bounds["unfollow"] = [2]int64{v.lo, v.hi}
		if v.hi > 30 {
			t.Errorf("%s/%s: unfollow acima de 30%% dos seguidores", sc.id, label)
		}
	case refundEffect:
		bounds["refund"] = [2]int64{v.lo, v.hi}
	}
	for name, b := range bounds {
		if b[0] <= 0 || b[1] < b[0] {
			t.Errorf("%s/%s: %s com faixa inválida %v", sc.id, label, name, b)
		}
	}
}

func checkPrize(t *testing.T, sc listedScenario, label, id string) {
	t.Helper()
	it, ok := itemByID(id)
	if !ok {
		t.Errorf("%s/%s: item desconhecido %q", sc.id, label, id)
		return
	}
	if it.rarity >= mythic || it.luxury {
		t.Errorf("%s/%s: evento não pode dar mítico, relíquia, primordial nem luxo (%s)", sc.id, label, id)
	}
	if (it.relicOdds > 0 || it.gear != nil && it.gear.tool) && sc.rarity == commonEvent {
		t.Errorf("%s/%s: só evento raro ou lendário dá ferramenta ou caixa de relíquia (%s)", sc.id, label, id)
	}
}

func checkText(t *testing.T, id, text string, victimOK bool) {
	t.Helper()
	if strings.TrimSpace(text) == "" || strings.Contains(text, "—") || strings.Contains(text, "%[") {
		t.Errorf("%s: texto vazio, com travessão ou com verbo de formatação: %q", id, text)
	}
	for _, p := range placeholder.FindAllString(text, -1) {
		if p != "{@}" && (p != "{alvo}" || !victimOK) {
			t.Errorf("%s: marcador %s não vale aqui: %q", id, p, text)
		}
	}
}

func TestEventEffects(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000000", types.GroupServer)
	acc := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	victim := core.Account{Key: "5511900000002@s.whatsapp.net", JID: types.NewJID("5511900000002", types.DefaultUserServer)}
	job, _ := professionNamed("bico")

	reset := func(t *testing.T, uses string) {
		t.Helper()
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			for _, a := range []core.Account{acc, victim} {
				if err := openWallet(ctx, tx, chat, a); err != nil {
					return err
				}
			}
			if _, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = 10000, morto_ate = 0, preso_ate = 0 WHERE chat = ?", chat.String()); err != nil {
				return err
			}
			for _, id := range []string{"vara-de-pesca", "picareta", "rede-de-insetos", uses} {
				if _, ok := itemByID(id); ok {
					if err := addItems(ctx, tx, chat, acc, id, 1); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	all := []effect{
		earn(50, 150), lose(5, 10), gain(2, "baseado"), forfeit("baseado", 1), findLoot(), injure(5, 10), heal(5),
		buzz("baseado"), nabbed(), stain(), enchant("sorte"), enchant("azar"), enchant("purificacao"), grudge("dizimo"), grudge("assombracao"), refresh(), delay(time.Minute),
		breakTool(), plant("teste", time.Hour), refund(5, 10), fans(5, 10), unfollow(5, 10), injure(200, 200), perish(),
	}
	for _, place := range eventPlaces {
		reset(t, "")
		for _, fx := range all {
			run := eventRun{chat: chat, acc: acc, victim: victim, job: job, place: place, stake: 100, now: time.Now()}
			if _, err := s.playEvent(ctx, run, eventPlay{ending: ending{effects: []effect{fx}}}); err != nil {
				t.Errorf("%s: efeito %T: %v", place, fx, err)
			}
			reset(t, "")
		}
	}

	for _, sc := range eventCatalog {
		place := "!trabalhar"
		if len(sc.places) > 0 {
			place = sc.places[0]
		}
		for _, ch := range sc.choices {
			for _, e := range ch.endings {
				reset(t, ch.uses)
				run := eventRun{chat: chat, acc: acc, victim: victim, job: job, place: place, stake: 100, now: time.Now()}
				res, err := s.playEvent(ctx, run, eventPlay{choice: &ch, price: ch.price.of(10000), ending: e})
				if err != nil {
					t.Errorf("%s/%s: %v", sc.id, ch.label, err)
				}
				if res.missing.id != "" || res.broke || res.gone {
					t.Errorf("%s/%s: não conseguiu jogar (%+v)", sc.id, ch.label, res)
				}
			}
		}
	}

	if err := s.noteEvent(ctx, chat, acc, "teste", time.Now()); err != nil {
		t.Fatal(err)
	}
	past, err := s.eventPast(ctx, chat, acc, time.Now().Add(2*time.Hour))
	if err != nil || !slices.Contains(past.recent, "teste") {
		t.Fatalf("eventPast: %v %+v", err, past)
	}
	seen, err := s.seenEvents(ctx, chat, acc)
	if err != nil || len(seen) != 1 {
		t.Fatalf("seenEvents: %v %+v", err, seen)
	}
}
