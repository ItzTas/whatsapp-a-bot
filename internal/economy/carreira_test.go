package economy

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
)

func TestCareerLevels(t *testing.T) {
	cases := map[time.Duration]int{
		0:                            0,
		careerFirst - time.Minute:    0,
		careerFirst:                  1,
		careerNeeds(2) - time.Minute: 1,
		careerNeeds(2):               2,
		careerNeeds(careerTop):       careerTop,
		careerNeeds(careerTop) * 10:  careerTop,
	}
	for worked, want := range cases {
		if got := (career{worked: worked}).level(); got != want {
			t.Errorf("com %s de trampo o nível é %d, deu %d", worked, want, got)
		}
	}
}

func TestSeasoned(t *testing.T) {
	bico := professions[0]
	job := bico.seasoned(map[string]time.Duration{bico.name: careerNeeds(2), "médico": careerNeeds(careerTop)})
	if job.min != bico.min*120/100 || job.max != bico.max*120/100 {
		t.Errorf("nível 2 paga +20%%: de %d-%d foi pra %d-%d", bico.min, bico.max, job.min, job.max)
	}
	if !strings.Contains(job.Details(), "nível 2") {
		t.Errorf("a carreira aparece nos detalhes da profissão:\n%s", job.Details())
	}
	if divine := god.seasoned(map[string]time.Duration{god.name: careerNeeds(careerTop)}); divine.min != god.min || divine.career.job != "" {
		t.Errorf("deus não tem carreira: %+v", divine.career)
	}
}

func TestCareerStore(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	acc := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}

	start := career{job: "bico"}
	half, err := s.build(ctx, chat, acc, start, careerFirst/2)
	if err != nil || half.worked != careerFirst/2 {
		t.Fatalf("meio caminho pro nível 1: %+v (%v)", half, err)
	}
	if got := start.progress(half); !strings.Contains(got, "+1h 30min de experiência como bico") || !strings.Contains(got, "Falta 1h 30min pro nível 1") {
		t.Errorf("mostra a experiência ganha e quanto falta: %q", got)
	}
	full, err := s.build(ctx, chat, acc, half, careerFirst/2)
	if err != nil || full.level() != 1 || !strings.Contains(half.progress(full), "Subiu pro nível 1") {
		t.Fatalf("o trampo soma e chega no nível 1: %+v (%v)", full, err)
	}
	if _, err := s.build(ctx, chat, acc, career{job: "médico"}, time.Hour); err != nil {
		t.Fatal(err)
	}
	worked, err := s.careers(ctx, chat, acc)
	if err != nil || worked["bico"] != careerFirst || worked["médico"] != time.Hour {
		t.Errorf("cada profissão guarda a própria carreira: %v (%v)", worked, err)
	}
}

func TestCareerDrillFits(t *testing.T) {
	named := func(name string) Profession {
		t.Helper()
		p, ok := professionNamed(name)
		if !ok {
			t.Fatalf("profissão %q não existe", name)
		}
		return p
	}
	if !named("pescador").crafts("!pescar") || named("pescador").crafts("!minerar") {
		t.Error("o pescador treina só no !pescar")
	}
	if named("bico").crafts("!pescar") || named("bico").farms() {
		t.Error("o bico não treina na extração nem na horta")
	}
	if !named("caçador de bússola").crafts("!catar") {
		t.Error("o caçador de bússola treina em todas as extrações")
	}
	if !named("fazendeiro").farms() {
		t.Error("o fazendeiro treina na horta")
	}
}

func TestDrilled(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenMessageStore(ctx, filepath.Join(t.TempDir(), "mensagens.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	s := &walletStore{db: &ledgerDB{db: store.DB}}
	chat := types.NewJID("120363000000000001", types.GroupServer)
	acc := core.Account{Key: "5511900000001@s.whatsapp.net", JID: types.NewJID("5511900000001", types.DefaultUserServer)}
	now := time.Unix(time.Now().Unix(), 0)
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		if err := openWallet(ctx, tx, chat, acc); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "UPDATE carteiras SET profissao = 'policial', prisao_em = ? WHERE chat = ? AND pessoa = ?", now.Unix(), chat.String(), acc.Key)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	job, done, err := s.drilled(ctx, chat, acc, "prisao_em", now.Add(-drillSlack))
	if err != nil || !done || job.name != "policial" || job.career.job != "policial" {
		t.Errorf("a prisão de agora conta pra carreira do policial: %s %v (%v)", job.name, done, err)
	}
	if _, done, err := s.drilled(ctx, chat, acc, "prisao_em", now.Add(time.Minute)); err != nil || done {
		t.Errorf("comando mandado depois da última prisão não conta: %v (%v)", done, err)
	}
}
