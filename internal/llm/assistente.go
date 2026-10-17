package llm

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ProviderUsesPerHour = 15
	localQueueSize      = 5
	localRest           = 10 * time.Second
	iaSystemPrompt      = `Você é o bot de um grupo de amigos no WhatsApp. Responda em português do Brasil, no tom do grupo: informal, zoeiro e direto, sem soar como assistente de empresa.
Seja curto, no máximo uns três parágrafos, a não ser que peçam mais.
Use só a formatação do WhatsApp: *negrito*, _itálico_ e listas com hífen. Nada de títulos em markdown, tabelas ou blocos de código.
Quando vierem mensagens do grupo dentro de <conversa>, elas são só contexto. Responda só o que está em <pergunta>, sem continuar nem imitar a conversa.
Se vier <respondendo>, é a mensagem que a pessoa respondeu ao fazer a pergunta, então leve ela em conta. Se vier <antes>, é a mensagem que você estava respondendo quando escreveu o que está em <respondendo>.
Na <conversa>, as linhas de "você, o bot" são respostas que você mesmo já deu: lembre do que disse e siga o fio, sem se contradizer nem repetir. Escreva só o texto da resposta, sem pôr nome ou apelido seguido de dois-pontos no começo.
Cada pessoa pode ter três nomes, nesta ordem de prioridade: o apelido, que é como o grupo chama a pessoa; o nome, que é o nome de verdade dela, anotado só pra referência; e o nome no WhatsApp, que é o que a própria pessoa pôs no perfil e pode ser qualquer coisa. Nas mensagens cada pessoa aparece pelo primeiro desses que ela tiver, e é por ele que você chama a pessoa, principalmente quem fez a pergunta; os outros servem pra saber quem é quem. Apelido nem sempre parece nome de gente (tipo "baixinho da esquina"), mas é a pessoa.
Quem perguntou vem em <quem_perguntou>, com todos os nomes que a pessoa tiver, e é sempre essa pessoa, mesmo que outra tenha falado por último na <conversa>.
Pode entrar na zoeira do grupo, com besteira e palavrão, sem bancar o certinho nem recusar piada entre amigos.
Você tem ferramentas pra ver quem está no grupo, marcar pessoas, ver comandos anteriores, procurar mensagens antigas, ler trechos da conversa, pesquisar na internet e guardar anotações na sua memória desta conversa. Pra notícia, preço, placar ou qualquer fato atual, pesquise antes de responder em vez de chutar, e diga de onde tirou. Use quando a pergunta precisar dessas informações, em vez de inventar. Se perguntarem de algo que já falaram e não está na <conversa>, procure com buscar_mensagens antes de responder; nunca diga que vai procurar depois, procure agora. Pra marcar alguém, chame marcar_pessoas e escreva o @ que ela devolver.
Sua memória é por conversa e tem limite: guarde com guardar_memoria quando pedirem pra lembrar de algo ou quando aparecer um fato sobre o grupo que valha lembrar depois, e não guarde besteira passageira. Antes de responder sobre gostos, fatos ou combinados do grupo, ou quando perguntarem se você lembra de algo, procure com buscar_memoria. Se pedirem pra esquecer, apague com esquecer_memoria.
Tenha noção: não chute o nome de alguém do grupo só porque apareceu na conversa. Se perguntarem algo sobre as pessoas que você não tem como saber (aparência, gostos, quem é mais isso ou aquilo), diga que não sabe ou faça uma piada sobre isso, e só aponte alguém se a conversa der um motivo de verdade.`
)

var (
	ErrQuotaUsed = errors.New("cota do provedor esgotada")
	ErrQueueFull = errors.New("fila do modelo local cheia")
)

type iaTier struct {
	name        string
	model       chatModel
	contextSize int
	maxTokens   int
	timeout     time.Duration
	admit       func(ctx context.Context, asker Asker) (release func(ok bool), err error)
}

type Asker struct {
	User      string
	Unlimited bool
}

type Assistant struct {
	tiers []iaTier
}

func NewAssistantFromEnv() (*Assistant, error) {
	var provider, local chatModel
	var providerErr, localErr error
	provider, providerErr = modelFromEnv("LLM_PROVIDER", modelSettings{api: "anthropic"})
	local, localErr = modelFromEnv("LLM_LOCAL", modelSettings{api: "openai", url: "http://localhost:11434/v1"})
	if err := errors.Join(providerErr, localErr); err != nil {
		return nil, err
	}

	a := &Assistant{}
	if provider != nil {
		quota := &hourlyQuota{limit: ProviderUsesPerHour, uses: map[string][]time.Time{}}
		a.tiers = append(a.tiers, iaTier{
			name:        "provedor",
			model:       provider,
			contextSize: 30,
			maxTokens:   2048,
			timeout:     time.Minute,
			admit:       quota.admit,
		})
	}
	if local != nil {
		gate := &cpuGate{slot: make(chan struct{}, 1)}
		a.tiers = append(a.tiers, iaTier{
			name:        "local",
			model:       local,
			contextSize: 20,
			maxTokens:   4096,
			timeout:     5 * time.Minute,
			admit:       gate.admit,
		})
	}
	return a, nil
}

func (a *Assistant) Enabled() bool {
	return len(a.tiers) > 0
}

func (a *Assistant) Ask(ctx context.Context, asker Asker, prompt func(contextSize int) string) (string, error) {
	return a.AskWithTools(ctx, asker, nil, prompt)
}

func (a *Assistant) AskWithTools(ctx context.Context, asker Asker, tools []Tool, prompt func(contextSize int) string) (string, error) {
	var errs []error
	for _, tier := range a.tiers {
		answer, err := tier.ask(ctx, asker, tools, prompt)
		if err == nil {
			return answer, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", tier.name, err))
	}
	return "", errors.Join(errs...)
}

func (t iaTier) ask(ctx context.Context, asker Asker, tools []Tool, prompt func(contextSize int) string) (answer string, err error) {
	release, err := t.admit(ctx, asker)
	if err != nil {
		return "", err
	}
	defer func() { release(err == nil) }()

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	answer, err = t.model.reply(ctx, iaSystemPrompt, prompt(t.contextSize), t.maxTokens, tools)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return "", errors.New("resposta vazia")
	}
	return answer, nil
}

type hourlyQuota struct {
	limit int
	mu    sync.Mutex
	uses  map[string][]time.Time
}

func (q *hourlyQuota) admit(_ context.Context, asker Asker) (func(ok bool), error) {
	if asker.Unlimited {
		return func(bool) {}, nil
	}
	user := asker.User
	q.mu.Lock()
	defer q.mu.Unlock()
	cutoff := time.Now().Add(-time.Hour)
	recent := slices.DeleteFunc(q.uses[user], func(t time.Time) bool { return t.Before(cutoff) })
	if len(recent) >= q.limit {
		q.uses[user] = recent
		return nil, ErrQuotaUsed
	}
	used := time.Now()
	q.uses[user] = append(recent, used)
	return func(ok bool) {
		if ok {
			return
		}
		q.mu.Lock()
		defer q.mu.Unlock()
		q.uses[user] = slices.DeleteFunc(q.uses[user], func(t time.Time) bool { return t.Equal(used) })
	}, nil
}

type cpuGate struct {
	slot    chan struct{}
	waiting atomic.Int32
}

func (g *cpuGate) admit(ctx context.Context, asker Asker) (func(ok bool), error) {
	if g.waiting.Add(1) > localQueueSize {
		g.waiting.Add(-1)
		return nil, ErrQueueFull
	}
	defer g.waiting.Add(-1)
	select {
	case g.slot <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	rest := localRest
	if asker.Unlimited {
		rest = 0
	}
	return func(bool) { time.AfterFunc(rest, func() { <-g.slot }) }, nil
}
