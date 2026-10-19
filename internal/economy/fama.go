package economy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"golang.org/x/sync/errgroup"
)

const (
	baseReach       = 25
	artReach        = 100
	stageBonus      = 50
	botoReach       = 100
	sponsorBacklash = 0.15
	backlashMin     = 3
	backlashMax     = 10
)

var (
	postCooldown    = cooldown{column: "postagem_em", every: 30 * time.Minute}
	sponsorCooldown = cooldown{column: "publi_em", every: 6 * time.Hour}
)

type platform struct {
	key         string
	name        string
	emoji       string
	aliases     []string
	style       string
	base        [2]int64
	growth      float64
	ceiling     float64
	viral       float64
	viralMin    float64
	viralMax    float64
	viralGrowth float64
	flop        float64
	cancel      float64
	loss        [2]int64
	grace       time.Duration
	decay       float64
	sponsor     int64
	posts       []string
	virals      []string
	flops       []string
	cancels     []string
}

var platforms = []platform{
	{
		key: "tiktok", name: "TikTok", emoji: "🎵", aliases: []string{"tt", "tik"},
		style: "pouco seguidor por vídeo, mas é onde mais viraliza, e onde mais cancelam",
		base:  [2]int64{1, 8}, growth: 0.006, ceiling: 50e6,
		viral: 0.03, viralMin: 200, viralMax: 30_000, viralGrowth: 0.1,
		flop: 0.3, cancel: 0.02, loss: [2]int64{5, 20},
		grace: 24 * time.Hour, decay: 0.03, sponsor: 8,
		posts: []string{
			"fez a dancinha da moda com a família toda", "postou um POV que ninguém entendeu", "dublou um áudio viral com a cara séria",
			"fez a trend de troca de roupa no espelho do banheiro", "gravou receita de miojo gourmet", "fez \"arrume-se comigo\" pra ir na padaria",
		},
		virals: []string{
			"fez uma dancinha tão ruim que virou meme no Brasil inteiro", "caiu da cadeira gravando e o vídeo rodou todos os grupos da família",
			"o vídeo foi parar no Para Você do mundo todo", "um famoso fez dueto com o vídeo",
		},
		flops: []string{
			"postou e o vídeo ficou com 12 visualizações, todas da mãe", "o algoritmo escondeu o vídeo de todo mundo",
			"usou um áudio com direitos autorais e o vídeo ficou mudo",
		},
		cancels: []string{
			"fez piada com quem não devia e foi cancelado(a) nos comentários", "acharam um vídeo antigo e o tribunal da internet não perdoou",
			"foi pego(a) comprando seguidor",
		},
	},
	{
		key: "youtube", name: "YouTube", emoji: "▶️", aliases: []string{"yt", "canal"},
		style: "cresce devagar e firme, quase não cancela e a publi paga bem",
		base:  [2]int64{2, 6}, growth: 0.012, ceiling: 30e6,
		viral: 0.008, viralMin: 150, viralMax: 8_000, viralGrowth: 0.06,
		flop: 0.2, cancel: 0.01, loss: [2]int64{5, 15},
		grace: 24 * time.Hour, decay: 0.03, sponsor: 15,
		posts: []string{
			"gravou \"reagindo ao meu primeiro vídeo\"", "fez um tutorial de 40 minutos que podia ser de 2", "postou vlog indo no mercado",
			"fez thumbnail com seta vermelha e a boca aberta", "gravou \"24 horas dentro de um Uno\"", "fez review de um celular de 2014",
		},
		virals: []string{
			"entrou nos vídeos em alta", "fez um documentário de 3 horas sobre um meme e todo mundo assistiu inteiro",
			"um canal gringo reagiu ao vídeo",
		},
		flops: []string{
			"ficou no limbo do algoritmo", "levou strike por causa da música de fundo",
			"esqueceu de ligar o microfone e gravou 20 minutos mudo",
		},
		cancels: []string{
			"pediu desculpas num vídeo com fundo cinza e piorou tudo", "fez publi escondida e os inscritos descobriram",
			"foi exposto(a) num vídeo de treta",
		},
	},
	{
		key: "instagram", name: "Instagram", emoji: "📸", aliases: []string{"insta", "ig", "reels"},
		style: "meio-termo em tudo, e é onde a publi paga melhor",
		base:  [2]int64{1, 6}, growth: 0.009, ceiling: 40e6,
		viral: 0.015, viralMin: 150, viralMax: 15_000, viralGrowth: 0.08,
		flop: 0.25, cancel: 0.015, loss: [2]int64{5, 20},
		grace: 24 * time.Hour, decay: 0.03, sponsor: 20,
		posts: []string{
			"postou foto do café da manhã que esfriou enquanto tirava 80 fotos", "fez carrossel de \"10 coisas que aprendi aos 25\"",
			"postou story na academia sem treinar", "postou foto do pé na areia com frase motivacional", "fez um reels de \"um dia na minha vida\"",
		},
		virals: []string{
			"o reels bombou no explorar", "uma página de fofoca repostou a foto", "a foto virou meme de \"expectativa vs realidade\"",
		},
		flops: []string{
			"a foto ficou com 3 curtidas, uma de robô", "o filtro deixou a cara laranja e ninguém curtiu", "postou às 4 da manhã e ninguém viu",
		},
		cancels: []string{
			"postou foto com filtro e o antes e depois vazou", "fez sorteio e esqueceu de entregar o prêmio",
			"foi pego(a) curtindo foto antiga do ex às 3 da manhã",
		},
	},
	{
		key: "twitch", name: "Twitch", emoji: "🎮", aliases: []string{"live", "lives", "stream"},
		style: "cresce firme, mas quem fica sem fazer live perde seguidor rápido",
		base:  [2]int64{3, 9}, growth: 0.01, ceiling: 10e6,
		viral: 0.005, viralMin: 100, viralMax: 5_000, viralGrowth: 0.05,
		flop: 0.2, cancel: 0.01, loss: [2]int64{5, 15},
		grace: 12 * time.Hour, decay: 0.06, sponsor: 12,
		posts: []string{
			"fez live de 8 horas jogando Minecraft", "fez live só conversando com o chat", "jogou um jogo de terror gritando a cada 5 minutos",
			"dormiu ao vivo e ainda ganhou sub", "fez maratona de react",
		},
		virals: []string{
			"levou raid de um streamer gigante", "deu um surto ao vivo e o clipe rodou a internet",
			"ganhou uma skin rara ao vivo e o chat explodiu",
		},
		flops: []string{
			"ficou 3 horas ao vivo com 0 espectadores", "a internet caiu no meio da live", "esqueceu a câmera virada e transmitiu 2 horas de teto",
		},
		cancels: []string{
			"falou besteira ao vivo e tomou ban de 7 dias", "foi pego(a) usando hack na live", "xingou o chat e o clipe vazou",
		},
	},
}

type fameTier struct {
	min     int64
	name    string
	emoji   string
	ransom  int64
	escape  float64
	snitch  float64
	fee     int64
	ceiling int64
	brands  []string
}

var fameTiers = []fameTier{
	{min: 0, name: "anônimo(a)", emoji: "👤"},
	{min: 1_000, name: "conhecido(a) no bairro", emoji: "🏘️", ransom: 10, escape: 0.02, snitch: 0.03, fee: 150, ceiling: 400,
		brands: []string{"da padaria do bairro", "do açougue da esquina", "da manicure da rua de baixo", "do lava-jato do primo"}},
	{min: 10_000, name: "microinfluencer", emoji: "📱", ransom: 25, escape: 0.04, snitch: 0.06, fee: 400, ceiling: 1_200,
		brands: []string{"de um chá detox", "de uma loja de capinha de celular", "de um curso de inglês em 30 dias", "de um whey vegano"}},
	{min: 100_000, name: "subcelebridade", emoji: "📢", ransom: 50, escape: 0.07, snitch: 0.1, fee: 1_500, ceiling: 3_500,
		brands: []string{"de uma casa de aposta", "de um aplicativo de delivery", "de uma cerveja artesanal", "de um banco digital pequeno"}},
	{min: 1_000_000, name: "famoso(a)", emoji: "🌟", ransom: 100, escape: 0.1, snitch: 0.15, fee: 4_000, ceiling: 8_000,
		brands: []string{"de uma montadora de carro", "de uma operadora de celular", "de uma rede gigante de fast food", "de uma marca de tênis"}},
	{min: 10_000_000, name: "celebridade", emoji: "👑", ransom: 200, escape: 0.15, snitch: 0.25, fee: 8_000, ceiling: 15_000,
		brands: []string{"no intervalo da novela", "de um perfume com o seu nome", "da sua própria marca de roupa", "da Copa do Mundo"}},
}

type feed struct {
	platform  platform
	followers int64
	posted    time.Time
}

type audience []feed

type clip struct {
	scene    string
	gain     int64
	viral    bool
	flopped  bool
	canceled bool
	before   int64
	after    int64
	wait     time.Duration
}

func (t fameTier) famous() bool {
	return t.min > 0
}

func (a audience) total() int64 {
	var total int64
	for _, f := range a {
		total += f.followers
	}
	return total
}

func (a audience) sponsorship(tier fameTier) int64 {
	pay := float64(tier.fee)
	for _, f := range a {
		pay += float64(f.followers*f.platform.sponsor) / 1000
	}
	return min(int64(pay*(0.8+rand.Float64()*0.4)), tier.ceiling)
}

func platformNamed(query string) (platform, bool) {
	q := normalizeItemName(query)
	i := slices.IndexFunc(platforms, func(pl platform) bool { return q == pl.key || slices.Contains(pl.aliases, q) })
	if i < 0 {
		return platform{}, false
	}
	return platforms[i], true
}

func (pl platform) devices() []item {
	return slices.DeleteFunc(slices.Clone(catalog), func(it item) bool { return !slices.Contains(it.networks, pl.key) })
}

func (pl platform) deviceNames() string {
	devices := pl.devices()
	labels := make([]string, len(devices))
	for i, it := range devices {
		labels[i] = it.label()
	}
	return core.JoinWithOr(labels)
}

func (pl platform) postableFrom(held []item) bool {
	return slices.ContainsFunc(held, func(it item) bool { return slices.Contains(it.networks, pl.key) })
}

func randomPostable(held []item) (platform, bool) {
	postable := slices.DeleteFunc(slices.Clone(platforms), func(pl platform) bool { return !pl.postableFrom(held) })
	if len(postable) == 0 {
		return platform{}, false
	}
	return core.PickOne(postable), true
}

func networkNames(keys []string) string {
	names := make([]string, len(keys))
	for i, key := range keys {
		pl, _ := platformNamed(key)
		names[i] = "no " + pl.name
	}
	return core.JoinWithAnd(names)
}

func tierIndex(followers int64) int {
	i := len(fameTiers) - 1
	for i > 0 && followers < fameTiers[i].min {
		i--
	}
	return i
}

func (pl platform) faded(followers int64, posted, now time.Time) int64 {
	idle := now.Sub(posted) - pl.grace
	if followers == 0 || idle <= 0 {
		return followers
	}
	return int64(float64(followers) * math.Pow(1-pl.decay, idle.Hours()/24))
}

func (pl platform) room(followers int64) float64 {
	return max(0, 1-math.Log10(float64(followers)+1)/math.Log10(pl.ceiling))
}

func (pl platform) cancelOdds(job Profession) float64 {
	if job.charming {
		return 0
	}
	return pl.cancel
}

func (pl platform) roll(followers int64, job Profession) clip {
	luck, cancel := rand.Float64(), pl.cancelOdds(job)
	if luck < cancel && followers > 0 {
		cut := pl.loss[0] + rand.Int64N(pl.loss[1]-pl.loss[0]+1)
		return clip{scene: core.PickOne(pl.cancels), gain: -followers * cut / 100, canceled: true}
	}
	if luck < cancel+pl.flop {
		return clip{scene: core.PickOne(pl.flops), flopped: true}
	}
	room, crowd := pl.room(followers), float64(followers)
	gain := float64(pl.base[0]+rand.Int64N(pl.base[1]-pl.base[0]+1)) + crowd*rand.Float64()*pl.growth*room
	c := clip{scene: core.PickOne(pl.posts)}
	if rand.Float64() < pl.viral {
		gain += logUniform(pl.viralMin, pl.viralMax) + crowd*rand.Float64()*pl.viralGrowth*room
		c.scene, c.viral = core.PickOne(pl.virals), true
	}
	c.gain = int64(math.Round(gain * float64(job.reachOn(pl)) / 100))
	return c
}

func logUniform(low, high float64) float64 {
	return math.Exp(math.Log(low) + rand.Float64()*(math.Log(high)-math.Log(low)))
}

func (p Profession) famed(fans audience) Profession {
	p.fans = fans
	return p
}

func (p Profession) followers() int64 {
	return p.fans.total()
}

func (p Profession) celebrity() fameTier {
	return fameTiers[tierIndex(p.followers())]
}

func (p Profession) backlashOdds() float64 {
	if p.charming {
		return 0
	}
	return sponsorBacklash
}

func (p Profession) reachPercent() int64 {
	if p.reach > 0 {
		return p.reach
	}
	if p.area == areaArte {
		return artReach
	}
	return baseReach
}

func (p Profession) reachOn(pl platform) int64 {
	if p.stage != pl.key {
		return p.reachPercent()
	}
	return p.reachPercent() * (100 + stageBonus) / 100
}

func multiplier(percent int64) string {
	return strings.ReplaceAll(strconv.FormatFloat(float64(percent)/100, 'f', -1, 64), ".", ",") + "x"
}

func followerCount(n int64) string {
	if n == 1 {
		return "1 seguidor"
	}
	return core.Thousands(n) + " seguidores"
}

func (t fameTier) toll() string {
	return fmt.Sprintf("quem te sequestra te vende %d%% mais caro no !venderrefem, a polícia acredita %d%% mais no !denunciar contra você e o !fugir da cadeia (e a fuga do !assalto que deu errado) tem %d%% menos chance",
		t.ransom, percent(t.snitch), percent(t.escape))
}

func famePerks(p Profession) []string {
	var lines []string
	if i := slices.IndexFunc(platforms, func(pl platform) bool { return pl.key == p.stage }); i >= 0 {
		lines = append(lines, fmt.Sprintf("📲 Alcança %s no !postar (o normal é %s), e %s no %s, a rede da profissão", multiplier(p.reachPercent()), multiplier(baseReach), multiplier(p.reachOn(platforms[i])), platforms[i].name))
	}
	if p.stage == "" && p.reachPercent() != baseReach {
		lines = append(lines, fmt.Sprintf("📲 Alcança %s no !postar (o normal é %s)", multiplier(p.reachPercent()), multiplier(baseReach)))
	}
	if p.charming {
		lines = append(lines, "🐬 Encanta todo mundo: nenhum post é cancelado no !postar e a galera nunca acha a !publi forçada")
	}
	if t := p.celebrity(); t.famous() {
		lines = append(lines, fmt.Sprintf("%s É %s, com %s: faz !publi, mas %s", t.emoji, t.name, followerCount(p.followers()), t.toll()))
	}
	return lines
}

func fameLine(job Profession) string {
	if job.followers() == 0 {
		return ""
	}
	t := job.celebrity()
	return fmt.Sprintf("%s Fama: %s, com %s", t.emoji, t.name, followerCount(job.followers()))
}

func fameRansomText(hostageJob Profession, user string) string {
	t := hostageJob.celebrity()
	if t.ransom == 0 {
		return ""
	}
	return fmt.Sprintf("\n%s @%s é %s (%s), e famoso vale %d%% a mais no mercado clandestino", t.emoji, user, t.name, followerCount(hostageJob.followers()), t.ransom)
}

func recognizedText(job Profession, user string) string {
	t := job.celebrity()
	if t.escape == 0 {
		return ""
	}
	return fmt.Sprintf("\n📸 @%s é %s e todo mundo reconhece: fugir tem %d%% menos chance", user, t.name, percent(t.escape))
}

func spottedText(suspectJob Profession, user string) string {
	t := suspectJob.celebrity()
	if t.snitch == 0 {
		return ""
	}
	return fmt.Sprintf("\n📸 @%s é %s, e a polícia leva denúncia contra famoso mais a sério: +%d%% de chance", user, t.name, percent(t.snitch))
}

func tierShiftText(before, after int64, user string) string {
	from, to := tierIndex(before), tierIndex(after)
	if to == from {
		return ""
	}
	t := fameTiers[to]
	if to < from {
		return fmt.Sprintf("\n📉 @%s caiu pra %s *%s*", user, t.emoji, t.name)
	}
	return fmt.Sprintf("\n🎉 @%s agora é %s *%s*! Já dá pra fazer !publi, mas fama tem preço: veja no !fama o que muda", user, t.emoji, t.name)
}

func platformNames() string {
	names := make([]string, len(platforms))
	for i, pl := range platforms {
		names[i] = pl.name
	}
	return core.JoinWithOr(names)
}

func fameTierList() string {
	tiers := make([]string, len(fameTiers))
	for i, t := range fameTiers {
		tiers[i] = fmt.Sprintf("%s %s (%s)", t.emoji, t.name, core.Thousands(t.min))
	}
	return core.JoinWithAnd(tiers)
}

func postUsage() string {
	lines := []string{fmt.Sprintf("📲 Use assim: !postar <rede>, ou só !postar pra cair numa rede aleatória entre as que você tem aparelho. Dá pra postar a cada %s, numa rede por vez, e cada rede pede um aparelho na mochila (tem na !loja):", core.FormatWait(postCooldown.every))}
	for _, pl := range platforms {
		lines = append(lines, fmt.Sprintf("%s *%s* (!postar %s, com %s): %s", pl.emoji, pl.name, pl.key, pl.deviceNames(), pl.style))
	}
	lines = append(lines, "", fmt.Sprintf("_Quanto rende depende da profissão: quem é de Arte e mídia alcança mais gente, e youtuber, streamer, influencer e mais alguns ainda têm a rede deles, onde rende %d%% a mais (veja !profissao). Rede parada perde seguidor_", stageBonus))
	return strings.Join(lines, "\n")
}

func (c clip) text(pl platform, user string) string {
	if c.canceled && c.gain == 0 {
		return fmt.Sprintf("🚫 No %s, @%s %s, mas tem tão pouco seguidor que ninguém nem notou", pl.name, user, c.scene)
	}
	if c.canceled {
		return fmt.Sprintf("🚫 No %s, @%s %s. Perdeu %s", pl.name, user, c.scene, followerCount(-c.gain))
	}
	if c.viral {
		return fmt.Sprintf("🔥 VIRALIZOU! No %s, @%s %s: +%s", pl.name, user, c.scene, followerCount(c.gain))
	}
	if c.flopped {
		return fmt.Sprintf("😶 No %s, @%s %s. Ninguém novo seguiu", pl.name, user, c.scene)
	}
	if c.gain == 0 {
		return fmt.Sprintf("😶 No %s, @%s %s, mas ninguém novo seguiu", pl.name, user, c.scene)
	}
	return fmt.Sprintf("%s No %s, @%s %s e ganhou %s", pl.emoji, pl.name, user, c.scene, followerCount(c.gain))
}

func (b *bot) postar(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	args := core.PlainArgs(core.MessageText(evt.Message))
	pl, ok := platform{}, true
	if len(args) > 0 {
		pl, ok = platformNamed(args[0])
	}
	if !ok {
		return b.SendText(ctx, chat, "Não conheço essa rede 🤔\n\n"+postUsage())
	}
	acc := b.Account(ctx, person)
	var (
		job  Profession
		held []item
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { job, err = b.ProfessionOf(gctx, chat, acc); return err })
	g.Go(func() (err error) { held, err = b.Wallets.devices(gctx, chat, acc); return err })
	if err := g.Wait(); err != nil {
		return fmt.Errorf("postar: %w", err)
	}
	if len(args) == 0 {
		pl, ok = randomPostable(held)
	}
	if !ok {
		text := fmt.Sprintf("📵 @%s não tem como postar: precisa de um aparelho na mochila (tem na !loja)\n\n%s", person.User, postUsage())
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	if !pl.postableFrom(held) {
		text := fmt.Sprintf("📵 @%s não tem como postar no %s: precisa de %s na mochila (tem na !loja)", person.User, pl.name, pl.deviceNames())
		_, err := b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	cd := cooldown{column: postCooldown.column, every: postEvery(job)}
	c, err := b.Wallets.post(ctx, chat, acc, cd, pl, job, time.Now())
	if err != nil {
		return fmt.Errorf("postar: %w", err)
	}
	text := fmt.Sprintf("🎬 @%s ainda tá editando o último post. Dá pra postar de novo em %s", person.User, core.FormatWait(c.wait))
	if c.wait == 0 {
		total := job.followers() - c.before + c.after
		text = c.text(pl, person.User) + fmt.Sprintf("\n%s Agora são %s no %s (%s no total)", pl.emoji, core.Thousands(c.after), pl.name, core.Thousands(total)) + tierShiftText(job.followers(), total, person.User) + job.decrees.note(fameDecree)
	}
	_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
	if err != nil || c.wait > 0 {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!postar", won: !c.canceled && !c.flopped})
	return nil
}

func postEvery(job Profession) time.Duration {
	return job.hasted(postCooldown.every * time.Duration(100+job.decrees.level(fameDecree)) / 100)
}

func (b *bot) fama(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := core.FirstMentionOrSender(evt).ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	job, err := b.ProfessionOf(ctx, chat, acc)
	if err != nil {
		return err
	}
	var (
		postWait, sponsorWait time.Duration
		held                  []item
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { held, err = b.Wallets.devices(gctx, chat, acc); return err })
	g.Go(func() (err error) {
		postWait, err = b.Wallets.cooldownLeft(gctx, chat, acc, cooldown{column: postCooldown.column, every: postEvery(job)}, now)
		return err
	})
	g.Go(func() (err error) {
		sponsorWait, err = b.Wallets.cooldownLeft(gctx, chat, acc, cooldown{column: sponsorCooldown.column, every: job.hasted(sponsorCooldown.every)}, now)
		return err
	})
	if err := g.Wait(); err != nil {
		return fmt.Errorf("ler espera da fama: %w", err)
	}
	_, err = b.SendMentions(ctx, chat, fameReport(person, job, held, postWait, sponsorWait, now), []types.JID{person})
	return err
}

func fameReport(person types.JID, job Profession, held []item, postWait, sponsorWait time.Duration, now time.Time) string {
	i := tierIndex(job.followers())
	tier := fameTiers[i]
	lines := []string{
		fmt.Sprintf("🌟 *Fama de @%s*", person.User),
		fmt.Sprintf("%s Nível: *%s*, com %s no total", tier.emoji, tier.name, followerCount(job.followers())),
	}
	if i+1 < len(fameTiers) {
		next := fameTiers[i+1]
		lines = append(lines, fmt.Sprintf("_Próximo nível: %s %s, com %s_", next.emoji, next.name, followerCount(next.min)))
	}
	lines = append(lines, "")
	for _, f := range job.fans {
		lines = append(lines, feedLine(f, job, held, now))
	}
	lines = append(lines, "", waitLine("📲 Posta", postWait))
	if !tier.famous() {
		return strings.Join(append(lines, fmt.Sprintf("_Manda !postar <rede> pra ganhar seguidores. Com %s no total já dá pra fazer !publi_", followerCount(fameTiers[1].min))), "\n")
	}
	lines = append(lines, waitLine("💼 Faz !publi", sponsorWait)+fmt.Sprintf(" (no seu nível, o cachê vai até %s)", Coins(tier.ceiling)), "", "⚠️ Fama tem preço: "+tier.toll())
	return strings.Join(lines, "\n")
}

func feedLine(f feed, job Profession, held []item, now time.Time) string {
	line := fmt.Sprintf("%s %s: %s · alcance %s", f.platform.emoji, f.platform.name, core.Thousands(f.followers), multiplier(job.reachOn(f.platform)))
	if job.stage == f.platform.key {
		line += " (a rede da profissão)"
	}
	if !f.platform.postableFrom(held) {
		line += fmt.Sprintf("\n   📵 Sem %s na mochila, não dá pra postar aqui", f.platform.deviceNames())
	}
	if f.followers == 0 {
		return line
	}
	idle := now.Sub(f.posted)
	if idle >= f.platform.grace {
		return line + fmt.Sprintf("\n   📉 Parada faz %s: perdendo %d%% por dia", core.FormatWait(idle), percent(f.platform.decay))
	}
	return line + fmt.Sprintf("\n   ⏳ Começa a perder seguidor em %s sem postar", core.FormatWait(f.platform.grace-idle))
}

func waitLine(action string, wait time.Duration) string {
	if wait > 0 {
		return fmt.Sprintf("%s de novo em %s", action, core.FormatWait(wait))
	}
	return action + " agora mesmo"
}

func (b *bot) publi(ctx context.Context, evt *events.Message) error {
	chat := evt.Info.Chat.ToNonAD()
	person := evt.Info.Sender.ToNonAD()
	acc := b.Account(ctx, person)
	now := time.Now()
	job, p, err := b.workTerms(ctx, chat, acc, now)
	if err != nil {
		return err
	}
	tier := job.celebrity()
	if !tier.famous() {
		text := fmt.Sprintf("📵 Nenhuma marca quer fazer publi com @%s, que tem %s. Chega em %s no total com o !postar e volta aqui", person.User, followerCount(job.followers()), core.Thousands(fameTiers[1].min))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	pay := job.fans.sponsorship(tier)
	var backlash int64
	if rand.Float64() < job.backlashOdds() {
		backlash = backlashMin + rand.Int64N(backlashMax-backlashMin+1)
	}
	levy := p.incomeLevy(acc)
	cd := cooldown{column: sponsorCooldown.column, every: job.hasted(sponsorCooldown.every)}
	w, err := b.Wallets.sponsor(ctx, chat, acc, cd, pay, backlash, levy, now)
	if err != nil {
		return fmt.Errorf("fazer publi: %w", err)
	}
	if w.wait > 0 {
		text := fmt.Sprintf("🤝 @%s fez uma publi faz pouco tempo. A próxima marca só liga em %s", person.User, core.FormatWait(w.wait))
		_, err = b.SendMentions(ctx, chat, text, []types.JID{person})
		return err
	}
	text := fmt.Sprintf("💼 @%s fez publi %s pros %s e recebeu %s", person.User, core.PickOne(tier.brands), followerCount(job.followers()), Coins(pay)) + incomeTaxText(w.tax, levy) + w.tithe.text()
	if backlash > 0 {
		left := job.followers() * (100 - backlash) / 100
		text += fmt.Sprintf("\n😡 Só que a galera achou forçado: @%s perdeu %d%% dos seguidores em todas as redes", person.User, backlash) + tierShiftText(job.followers(), left, person.User)
	}
	people := []types.JID{person}
	if w.tax > 0 {
		people = append(people, levy.payee()...)
	}
	if _, err := b.SendMentions(ctx, chat, text, people); err != nil {
		return err
	}
	b.stirEvent(ctx, evt, eventSpot{place: "!publi", won: backlash == 0})
	return nil
}

func (s *walletStore) audience(ctx context.Context, chat types.JID, acc core.Account, now time.Time) (a audience, err error) {
	rows, err := s.db.QueryContext(ctx, "SELECT plataforma, seguidores, postado_em FROM fama WHERE chat = ? AND pessoa = ?", chat.String(), acc.Key)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	stored := make(map[string]feed)
	for rows.Next() {
		var (
			key               string
			followers, posted int64
		)
		if err := rows.Scan(&key, &followers, &posted); err != nil {
			return nil, err
		}
		stored[key] = feed{followers: followers, posted: time.Unix(posted, 0)}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	a = make(audience, len(platforms))
	for i, pl := range platforms {
		f := stored[pl.key]
		a[i] = feed{platform: pl, followers: pl.faded(f.followers, f.posted, now), posted: f.posted}
	}
	return a, nil
}

func (s *walletStore) devices(ctx context.Context, chat types.JID, acc core.Account) ([]item, error) {
	return s.holding(ctx, chat, acc, func(it item) bool { return it.networks != nil })
}

func (s *walletStore) post(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, pl platform, job Profession, now time.Time) (c clip, err error) {
	err = s.inTx(ctx, func(tx *ledgerTx) error {
		wait, err := stampCooldown(ctx, tx, chat, acc, cd, now)
		if err != nil || wait > 0 {
			c.wait = wait
			return err
		}
		var followers, posted int64
		err = tx.QueryRowContext(ctx,
			"SELECT seguidores, postado_em FROM fama WHERE chat = ? AND pessoa = ? AND plataforma = ?",
			chat.String(), acc.Key, pl.key,
		).Scan(&followers, &posted)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		before := pl.faded(followers, time.Unix(posted, 0), now)
		c = pl.roll(before, job)
		c.before, c.after = before, max(before+c.gain, 0)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO fama (chat, pessoa, plataforma, seguidores, postado_em) VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT (chat, pessoa, plataforma) DO UPDATE SET seguidores = excluded.seguidores, postado_em = excluded.postado_em`,
			chat.String(), acc.Key, pl.key, c.after, now.Unix(),
		)
		return err
	})
	return c, err
}

func (s *walletStore) sponsor(ctx context.Context, chat types.JID, acc core.Account, cd cooldown, amount, backlash int64, levy PixLevy, now time.Time) (workday, error) {
	w, err := s.claim(ctx, chat, acc, cd, amount, item{}, levy, now)
	if err != nil || w.wait > 0 || backlash == 0 {
		return w, err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE fama SET seguidores = seguidores * ? / 100 WHERE chat = ? AND pessoa = ?", 100-backlash, chat.String(), acc.Key)
	return w, err
}
