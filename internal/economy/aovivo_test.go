package economy

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow/types"
)

func liveListing(id string, rarity eventRarity) listedScenario {
	return listedScenario{id: id, rarity: rarity}
}

func eventIDs() map[string]int {
	ids := map[string]int{}
	for _, sc := range eventCatalog {
		ids[sc.id]++
	}
	return ids
}

func checkLiveEnding(t *testing.T, sc listedScenario, label string, e ending, people bool) {
	t.Helper()
	if len(e.texts) == 0 {
		t.Errorf("%s/%s: desfecho sem texto", sc.id, label)
	}
	if e.favor != nil || e.next != "" {
		t.Errorf("%s/%s: evento ao vivo não usa favor nem next", sc.id, label)
	}
	checkLiveEffects(t, sc, label, e.effects)
	for _, text := range e.texts {
		checkLiveText(t, sc.id, text, people)
	}
}

func checkLiveEffects(t *testing.T, sc listedScenario, label string, effects []effect) {
	t.Helper()
	for _, fx := range effects {
		switch fx.(type) {
		case breakEffect, refundEffect:
			t.Errorf("%s/%s: %T não faz sentido em evento ao vivo", sc.id, label, fx)
		}
		checkEffect(t, sc, label, fx, eventIDs(), map[string]bool{})
	}
}

func checkLiveText(t *testing.T, id, text string, people bool) {
	t.Helper()
	checkText(t, id, text, false)
	if !people && placeholder.MatchString(text) {
		t.Errorf("%s: texto coletivo não pode marcar ninguém: %q", id, text)
	}
}

func TestLiveCatalog(t *testing.T) {
	ids := map[string]int{}
	for _, e := range liveCatalog {
		ids[e.id]++
	}
	for id, n := range ids {
		if id == "" || n > 1 {
			t.Errorf("%q: id de evento ao vivo vazio ou repetido", id)
		}
	}
	for _, sc := range grabScenarios {
		checkGrab(t, sc)
	}
	for _, sc := range bossScenarios {
		checkBoss(t, sc)
	}
	for _, sc := range voteScenarios {
		checkVote(t, sc)
	}
}

func checkGrab(t *testing.T, sc grabScenario) {
	t.Helper()
	listed := liveListing(sc.id, sc.rarity)
	if sc.title == "" || len(sc.intros) == 0 || len(sc.missed) == 0 || len(sc.endings) == 0 {
		t.Errorf("%s: falta título, abertura, desfecho ou texto de ninguém pegou", sc.id)
	}
	if (sc.lock == "") != (sc.riddle == nil) {
		t.Errorf("%s: senha precisa de lock e riddle juntos", sc.id)
	}
	if sc.riddle != nil {
		clue, answer := sc.riddle()
		if answer == "" || !sameAnswer(answer, answer) || sameAnswer(clue, answer) {
			t.Errorf("%s: senha que não dá pra acertar ou que já vem respondida: %q -> %q", sc.id, clue, answer)
		}
	}
	for _, text := range slices.Concat(sc.intros, sc.missed) {
		checkLiveText(t, sc.id, text, false)
	}
	for _, e := range sc.endings {
		if e.weight <= 0 {
			t.Errorf("%s: desfecho sem peso", sc.id)
		}
		checkLiveEnding(t, listed, "corrida", e, true)
	}
}

func checkBoss(t *testing.T, sc bossScenario) {
	t.Helper()
	listed := liveListing(sc.id, sc.rarity)
	if sc.title == "" || len(sc.intros) == 0 || len(sc.hits) < 3 || sc.toughness <= 0 || len(sc.trophy) == 0 {
		t.Errorf("%s: falta título, abertura, golpes, vida ou troféu", sc.id)
	}
	if sc.counter.weight <= 0 || sc.counter.weight > 50 {
		t.Errorf("%s: chance do contra-ataque fora de 1 a 50%%: %d", sc.id, sc.counter.weight)
	}
	for _, text := range sc.intros {
		checkLiveText(t, sc.id, text, false)
	}
	for _, text := range sc.hits {
		checkLiveText(t, sc.id, text, true)
	}
	checkLiveEnding(t, listed, "contra-ataque", sc.counter, true)
	checkLiveEnding(t, listed, "vitória", sc.victory, true)
	checkLiveEnding(t, listed, "fuga", sc.defeat, false)
	checkLiveEffects(t, listed, "troféu", sc.trophy)
}

func checkVote(t *testing.T, sc voteScenario) {
	t.Helper()
	listed := liveListing(sc.id, sc.rarity)
	if sc.title == "" || len(sc.intros) == 0 || len(sc.idle) == 0 {
		t.Errorf("%s: falta título, abertura ou texto de ninguém votou", sc.id)
	}
	if len(sc.choices) < 2 || len(sc.choices) > 4 {
		t.Errorf("%s: precisa de 2 a 4 opções, tem %d", sc.id, len(sc.choices))
	}
	for _, text := range slices.Concat(sc.intros, sc.idle) {
		checkLiveText(t, sc.id, text, false)
	}
	labels := map[string]bool{}
	for _, ch := range sc.choices {
		if ch.label == "" || labels[ch.label] || utf8.RuneCountInString(ch.label) > 45 {
			t.Errorf("%s: opção vazia, repetida ou longa demais: %q", sc.id, ch.label)
		}
		labels[ch.label] = true
		if ch.needs != nil || ch.uses != "" || ch.price != (cost{}) || len(ch.endings) == 0 {
			t.Errorf("%s/%s: opção de enquete coletiva não tem needs, uses nem price, e precisa de desfecho", sc.id, ch.label)
		}
		checkVoteOdds(t, sc.id, ch)
		for _, e := range ch.endings {
			if e.weight <= 0 {
				t.Errorf("%s/%s: desfecho sem peso", sc.id, ch.label)
			}
			checkLiveEnding(t, listed, ch.label, e, false)
		}
	}
}

func checkVoteOdds(t *testing.T, id string, ch choice) {
	t.Helper()
	var total, deadly int
	for _, e := range ch.endings {
		total += e.weight
		if slices.ContainsFunc(e.effects, func(fx effect) bool { _, ok := fx.(perishEffect); return ok }) {
			deadly += e.weight
		}
	}
	if deadly > 0 && (ch.risk != highRisk || deadly*10 > total) {
		t.Errorf("%s/%s: morte só em opção de risco alto e em até 10%% das vezes", id, ch.label)
	}
}

func TestLiveEffects(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	acc := core.Account{Key: "5511900000003@s.whatsapp.net", JID: types.NewJID("5511900000003", types.DefaultUserServer)}
	job, _ := professionNamed("bico")

	play := func(t *testing.T, id string, effects []effect) {
		t.Helper()
		err := s.inTx(ctx, func(tx *ledgerTx) error {
			if err := openWallet(ctx, tx, chat, acc); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, "UPDATE carteiras SET saldo = 10000, morto_ate = 0, preso_ate = 0 WHERE chat = ?", chat.String())
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		run := eventRun{chat: chat, acc: acc, job: job, stake: 100, now: time.Now()}
		res, err := s.playEvent(ctx, run, eventPlay{ending: ending{effects: effects}})
		if err != nil || res.gone {
			t.Errorf("%s: %v (%+v)", id, err, res)
		}
	}

	for _, sc := range grabScenarios {
		for _, e := range sc.endings {
			play(t, sc.id, e.effects)
		}
	}
	for _, sc := range bossScenarios {
		play(t, sc.id, sc.counter.effects)
		play(t, sc.id, slices.Concat(sc.victory.effects, sc.trophy))
		play(t, sc.id, sc.defeat.effects)
	}
	for _, sc := range voteScenarios {
		for _, ch := range sc.choices {
			for _, e := range ch.endings {
				play(t, sc.id, e.effects)
			}
		}
	}
}

func TestLiveOdds(t *testing.T) {
	if liveOdds(0) != 0 || liveOdds(liveRest) != 0 {
		t.Errorf("evento ao vivo logo depois do último: %v %v", liveOdds(0), liveOdds(liveRest))
	}
	middle := liveRest + (liveDue-liveRest)/2
	if got := liveOdds(middle); got <= 0 || got >= livePeak || got <= liveOdds(liveRest+time.Minute) {
		t.Errorf("chance no meio da subida fora da faixa: %v", got)
	}
	if liveOdds(liveDue) != 1 || liveOdds(liveDue+time.Hour) != 1 {
		t.Errorf("depois do prazo o próximo comando precisa abrir: %v", liveOdds(liveDue))
	}
}

func TestLiveRoster(t *testing.T) {
	var board liveBoard
	chat := types.NewJID("120363000000000002", types.GroupServer)
	now := time.Now()
	old := core.Account{Key: "old", JID: types.NewJID("5511900000010", types.DefaultUserServer)}
	fresh := core.Account{Key: "fresh", JID: types.NewJID("5511900000011", types.DefaultUserServer)}
	board.cue(chat, old, now.Add(-livePlayerSpan-time.Hour))
	cue := board.cue(chat, fresh, now)
	if len(cue.players) != 1 || cue.players[0] != fresh.JID {
		t.Errorf("jogadores do dia: %v", cue.players)
	}
	if cue.save == nil {
		t.Error("jogador novo tem que salvar a lista")
	}
	board.opened(chat, "corrida-malote", now)
	if cue := board.cue(chat, fresh, now.Add(time.Minute)); cue.due || cue.save != nil || !slices.Contains(cue.recent, "corrida-malote") {
		t.Errorf("logo depois de abrir: %+v", cue)
	}
}

func TestPickLive(t *testing.T) {
	if formats := crowdFormats(1); !slices.Equal(formats, []liveFormat{grabFormat}) {
		t.Errorf("com pouca gente só sai corrida: %v", formats)
	}
	if formats := crowdFormats(liveCrowd); len(formats) != len(liveFormats) {
		t.Errorf("com gente suficiente sai qualquer formato: %v", formats)
	}
	recent := make([]string, 0, len(grabScenarios))
	for _, sc := range grabScenarios[1:] {
		recent = append(recent, sc.id)
	}
	for range 20 {
		entry, ok := pickLive([]liveFormat{grabFormat}, recent)
		if !ok || entry.id != grabScenarios[0].id {
			t.Fatalf("tem que sortear a corrida que ainda não saiu: %+v", entry)
		}
	}
	names := map[string]liveFormat{"chefão": bossFormat, "Enquete": voteFormat, "corrida": grabFormat}
	for name, want := range names {
		if got, ok := liveFormatsNamed(name); !ok || !slices.Equal(got, []liveFormat{want}) {
			t.Errorf("%q: %v %v", name, got, ok)
		}
	}
	if _, ok := liveFormatsNamed("baile"); ok {
		t.Error("tipo desconhecido não pode abrir evento")
	}
}

func TestBossStrike(t *testing.T) {
	sc := bossScenarios[0]
	ev := newLiveEvent(types.NewJID("120363000000000003", types.GroupServer), liveEntry{format: bossFormat, id: sc.id}, 1, time.Now())
	if ev.life != sc.toughness*liveCrowd || ev.health != ev.life {
		t.Fatalf("vida do chefão: %d/%d", ev.health, ev.life)
	}
	a := core.Account{Key: "a"}
	b := core.Account{Key: "b"}
	now := time.Now()
	if hit := ev.strike(a, 10, now); hit.dealt != 10 || hit.health != ev.life-10 || hit.felled {
		t.Errorf("primeiro golpe: %+v", hit)
	}
	if hit := ev.strike(a, 10, now.Add(time.Minute)); hit.wait != bossRest-time.Minute {
		t.Errorf("golpe antes da espera: %+v", hit)
	}
	hit := ev.strike(b, ev.life*2, now)
	if !hit.felled || hit.dealt != ev.life-10 || hit.health != 0 {
		t.Errorf("golpe final: %+v", hit)
	}
	if hit := ev.strike(a, 10, now.Add(bossRest)); !hit.gone {
		t.Errorf("chefão derrubado não apanha mais: %+v", hit)
	}
	ranking, _ := ev.ranking()
	if len(ranking) != 2 || ranking[0].acc.Key != "b" {
		t.Errorf("ranking por dano: %+v", ranking)
	}
}

func TestBossShare(t *testing.T) {
	shares := []struct {
		damage, total int64
		hitters       int
		want          int64
	}{
		{50, 100, 2, 100},
		{75, 100, 2, 150},
		{90, 100, 5, bossShareMax},
		{1, 100, 5, bossShareMin},
	}
	for _, s := range shares {
		if got := shareOf(s.damage, s.total, s.hitters); got != s.want {
			t.Errorf("shareOf(%d, %d, %d) = %d, queria %d", s.damage, s.total, s.hitters, got, s.want)
		}
	}
}

func TestMajority(t *testing.T) {
	ballots := []liveBallot{{Choice: 1}, {Choice: 1}, {Choice: 0}}
	if pick, votes := majority(ballots, 3); pick != 1 || votes != 2 {
		t.Errorf("maioria: %d com %d votos", pick, votes)
	}
	tied := []liveBallot{{Choice: 0}, {Choice: 2}}
	for range 20 {
		if pick, votes := majority(tied, 3); pick == 1 || votes != 1 {
			t.Fatalf("empate só sorteia entre as empatadas: %d com %d votos", pick, votes)
		}
	}
}

func TestSameAnswer(t *testing.T) {
	if !sameAnswer(" Naufrágio ", "naufragio") || !sameAnswer("5 9", "59") || sameAnswer("58", "59") {
		t.Error("resposta da senha ignora acento, maiúscula e espaço, e só aceita a certa")
	}
}
