package economy

import (
	"context"
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

const tipEvery = 3 * time.Hour

type tipBoard struct {
	mu    sync.Mutex
	decks map[types.JID]*tipDeck
}

type tipDeck struct {
	due  time.Time
	left []string
}

func (t *tipBoard) draw(chat types.JID, now time.Time, deal func() []string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.decks == nil {
		t.decks = map[types.JID]*tipDeck{}
	}
	d, ok := t.decks[chat]
	if !ok {
		t.decks[chat] = &tipDeck{due: now.Add(tipEvery)}
		return "", false
	}
	if now.Before(d.due) {
		return "", false
	}
	d.due = now.Add(tipEvery)
	if len(d.left) == 0 {
		d.left = deal()
		rand.Shuffle(len(d.left), func(i, j int) { d.left[i], d.left[j] = d.left[j], d.left[i] })
	}
	tip := d.left[len(d.left)-1]
	d.left = d.left[:len(d.left)-1]
	return tip, true
}

func (b *bot) offerTip(ctx context.Context, chat types.JID) {
	tip, ok := b.tips.draw(chat.ToNonAD(), time.Now(), b.allTips)
	if !ok {
		return
	}
	on, err := b.ChatModeOn(ctx, chat, core.ModeTips)
	if err != nil {
		b.Client.Log.Errorf("ver o modo dicas em %s: %v", chat, err)
		return
	}
	if !on {
		return
	}
	if err := b.SendText(ctx, chat, tipText(tip)); err != nil {
		b.Client.Log.Errorf("mandar dica em %s: %v", chat, err)
	}
}

func (b *bot) dica(ctx context.Context, evt *events.Message) error {
	topic := core.QuestionFrom(core.MessageText(evt.Message))
	folded := core.FoldLetters(topic)
	tips := slices.DeleteFunc(b.allTips(), func(t string) bool { return !strings.Contains(core.FoldLetters(t), folded) })
	if len(tips) == 0 {
		return b.ReplyTo(ctx, evt, fmt.Sprintf("não achei nenhuma dica sobre *%s* 🤔 tenta outra palavra, tipo `!dica relíquia` ou `!dica cadeia`", topic))
	}
	return b.ReplyTo(ctx, evt, tipText(tips[rand.IntN(len(tips))]))
}

func tipText(tip string) string {
	return fmt.Sprintf("💡 *Dica da economia*\n\n%s\n\n_Quer outra? Manda !dica (ou !dica <assunto>, tipo !dica relíquia)_", tip)
}

func (b *bot) allTips() []string {
	return slices.Concat(economyTips(), b.commandTips())
}

func (b *bot) commandTips() []string {
	var tips []string
	for _, c := range b.economyCommands() {
		var sb strings.Builder
		fmt.Fprintf(&sb, "Já conhece o *!%s*?", c.Name)
		for _, u := range c.Help {
			fmt.Fprintf(&sb, "\n\n▸ `%s`\n_%s_", u.Syntax, u.Description)
		}
		tips = append(tips, sb.String())
	}
	return tips
}

func economyTips() []string {
	return []string{
		"Quem saiu do grupo não escapa: no !roubar, !furtar, !sequestrar, !matar, !quadrilha, !prender e !denunciar dá pra mandar o nome da pessoa, o número dela (@5511999999999) ou responder uma mensagem ou um contato dela 🕵️",
		"Relíquia não sai de caixa comum. O único jeito é achar o *Relicário perdido* no !pescar, !minerar, !vasculhar ou !catar e abrir ele com !usar relicario 🏺",
		"A sorte da ferramenta e dos acessórios só ajuda a achar o *Relicário perdido* se você tiver a *Bússola do relicário* na mochila. Ela ainda aumenta a chance dele aparecer, mas é gasta a cada extração, então usa na hora certa 🧭",
		"A *Isca de minhoca* e a *Garrafa de melado* dão sorte, mas essa sorte nunca acha Relicário, nem com a bússola. E o *Ímã* tira o Relicário do sorteio: pra caçar relíquia, deixa ele fora da mochila 🧲",
		"A bênção *Faro* (Figa de guiné) dá um sorteio a mais na extração, mas esse sorteio ignora o Relicário. A *Achado* (Vela pra São Longuinho) aumenta a chance de tudo que é raro, Relicário incluído. Pra caçar relíquia, compra a vela na !loja e manda !usar 🕯️",
		"O *Banho de abre-caminho* (!usar) corta item comum e mão vazia nas extrações. Com menos lixo no sorteio, sobra mais espaço pros achados bons 🛁",
		"Mítico só aparece em extração com uma relíquia ou primordial equipado: a ferramenta relíquia da atividade, o Saco do Homem do Saco no !vasculhar, a Vela do Negrinho do Pastoreio (vale nas 4) ou o Meteorito do Bendegó. Sem isso ele nem entra no sorteio ✨",
		"O mercado do !vender paga bem mais por mítico do que pelo resto. Se a sua profissão já vende acima disso, vale o maior. Leiloeiro, penhorista e camelô vendem acima do normal 💸",
		"A *Rocha celeste* (primordial ⚫) só cai em extração feita com uma relíquia de extração equipada: Vara de Iemanjá, Picareta de Serra Pelada, Rede do Curupira ou Saco do Homem do Saco. A Vela do Negrinho não conta, e nenhuma sorte muda a chance ☄️",
		"Abrindo a *Rocha celeste* com !usar, às vezes vem um *Fragmento do Bendegó*. Os fragmentos viram um *Meteorito do Bendegó* sozinhos quando você junta o suficiente, venham de onde vierem, até de !dar ou !vender @pessoa. Dá pra fechar trocando com a galera, só não deixa guardado no imóvel, que lá ele não monta 🌠",
		"Quem tem o *Meteorito do Bendegó* traz 2 itens por extração e nunca volta de mão vazia, e o item puxado pela gravidade não gasta a ferramenta. Ele ainda abre a cratera com !primordial bendego uma vez por dia ☄️",
		"Viu alguém abrir a cratera do Bendegó? Corre pro !minerar (precisa de picareta): enquanto ela tá aberta sai mítico mesmo sem relíquia, e com sorte extra. Em troca, você paga um pedágio do que achar pro dono da cratera ⛏️",
		"Duas ferramentas da mesma atividade não somam: só vale a mais cara, e a outra fica parada na mochila. Já os acessórios (capacete, luva, lupa, bússola) somam junto com ela 🎣",
		"O !aprimorar leva uma ferramenta de extração rara, épica ou lendária no ferreiro, e ela volta achando coisa melhor. Você paga só a diferença de valor. Cada ferramenta aprimora uma vez só, e se ela quebrar o aprimoramento vai junto 🔨",
		"A qualidade muda o preço de venda: o mesmo item vale muito menos horrível e muito mais lendário. A *Lupa de joalheiro* da !loja e a bênção *Mão boa* melhoram a qualidade do que você acha nas extrações 🔍",
		"A *Vara de titânio*, a *Picareta de nióbio* e a *Rede de náilon* só aparecem no \"Só hoje\" da !loja, uma por dia em rodízio. As ferramentas lendárias nem ficam na loja: aparecem no !mercadonegro (quem comprar primeiro leva) ou no !leilao 🛒",
		"Comprar no !mercadonegro pode dar batida: você vai preso e perde a compra 🚔",
		"Muito de vez em quando o !leilao do dia traz uma *relíquia*. É o único jeito de comprar uma sem depender da sorte, mas o lance mínimo é salgado 🏺",
		"Sorte de caixa (youtuber de unboxing, *Estilete de unboxing*) melhora a raridade do prêmio comum das caixas, mas não muda a chance de relíquia do Relicário. Na *Rocha celeste*, ela aumenta a chance de mítico 📦",
		"Relíquia e primordial ninguém consegue !furtar, mas mítico pode ser furtado. Guarda os míticos em casa com !imovel guardar <item> até vender 🏠",
		"Antes de mandar !vender inuteis ou !vender tipo, usa o !favoritar no que quer manter: favorito não sai nessas vendas. Cuidado que o !vender duplicados ignora favorito e deixa só 1 de cada, a de melhor qualidade, até das bússolas e iscas que você ia gastar ⭐",
		"Precisa de *Bússola do relicário*? O caçador de bússola acha ela muito mais fácil e espera menos nas extrações. Em troca, tudo vem na pior qualidade e o !vender paga menos pra ele 🧭",
		"Cada tipo de bicho do !petshop tenta separado trazer um item no !daily, e bicho mais raro tem mais chance. Vários bichos diferentes ajudam, mas é no máximo um presente por dia 🐶",
		"Dinheiro no saldo é dinheiro exposto: ladrão, Leão do imposto, multa, perda em evento e penhora do banco só mexem no saldo. Quem tem imóvel guarda a reserva com !imovel depositar <valor>, que no cofre de casa ninguém encosta 🏦",
		"Com o !poupar, o dinheiro sai do saldo: lá ladrão não leva, o Leão não cobra e ainda rende juros. O resgate não cai sozinho, então manda !poupar de novo quando liberar e já guarda outra vez 🐷",
		"O Leão do !imposto só olha o saldo, e tem uma faixa isenta. Poupança, cofre de casa, !acoes e !conjunta não entram na conta 🦁",
		"Um contador no !empregados corta parte do imposto e do IPTU todo dia, sem salário e sem precisar coletar. Só vale o melhor que você tiver, então não adianta contratar dois 🧮",
		"Empregado que produz guarda só um limite de rodadas e depois para. Manda !empregados coletar com frequência e com saldo pro salário. Sem saldo, eles trabalham fiado e a tensão sobe, e quando ela chega no limite eles entram em greve e as rodadas se perdem até você pagar o atrasado 👷",
		"Quer renda estável no !empregados? Vai de *corretor*, que nunca volta com menos do que cobrou de salário. O *apostador* perde tudo na maioria das rodadas e só às vezes estoura 🎰",
		"Toda área de profissão tem um *instrumento de trabalho* na !loja. Só de estar na mochila, o !trabalhar paga mais e a chance de achar item sobe. Guardado no !imovel ele não vale 🧰",
		"Antes de trocar de profissão, manda !profissao <área> pra comparar quanto cada uma paga e o que tem de especial, ou !profissao ver <nome> pra ver uma só. A troca tem espera, então escolhe com calma 💼",
		"Imóvel e a doutrina Caridade da igreja aumentam o !daily, e bicho do !petshop às vezes traz um item de presente. Não esquece de pegar todo dia 🎁",
		"As !missoes são diferentes pra cada um e trocam à meia-noite, sem levar o progresso pro dia seguinte. Fechando todas, você ganha um bônus e uma caixa misteriosa, direto no saldo, sem dízimo nem imposto 🎯",
		"A missão de apostas só conta o *lucro* no !apostar e no !cassino. O !blackjack, o !crash e a !corrida não avançam ela, e perder não tira o que você já fez 🎲",
		"Nos eventos, o que você ganha acompanha quanto sua profissão paga no trampo, e o que você perde é uma porcentagem do saldo. Profissão boa e saldo baixo deixam os eventos bem mais vantajosos 📖",
		"Tem opção *secreta* nos eventos, que só aparece pra quem tem certo item na mochila ou é de certa profissão. Mochila variada abre mais saídas, e o !eventos mostra quantos você já descobriu 🤫",
		"Os *Búzios encantados* (relíquia) triplicam a chance de cair evento e mostram na enquete a chance de cada opção dar bom ou ruim 🐚",
		"Na enquete do evento só o seu voto vale, e dá pra responder a mensagem com o número da opção. Se demorar demais, você fica parado e aceita o que vier ⏳",
		"No !postar, cada rede cresce em cima dos seguidores *dela*, e a espera é uma só pra todas. Foca numa rede principal, de preferência a da sua profissão, que rende mais 📲",
		"Rede parada perde seguidor depois de um tempo sem post, e a Twitch esfria mais rápido que as outras. O !fama mostra quanto falta pra começar a cair 📉",
		"Quer ficar famoso? Profissão de *Arte e mídia* tem bem mais alcance no !postar, e algumas têm uma rede preferida que rende ainda mais. Confere no !profissao ver <nome> 🎬",
		"TikTok viraliza mais e também cancela mais, YouTube cresce firme e quase não cancela, e o Instagram é onde a !publi paga mais por seguidor. Celular libera TikTok e Instagram, Computador libera YouTube e Twitch 📱",
		"A !publi só libera depois que você sai do anonimato, tem teto de cachê por nível e pode dar ruim: a galera acha forçado e você perde seguidor em todas as redes. O *Chapéu do Boto* zera esse risco e os cancelamentos do !postar 🎩",
		"Fama tem preço: quanto mais famoso, mais caro fica seu resgate num sequestro, mais difícil fugir da cadeia e mais fácil alguém te !denunciar ⭐",
		"Antes de entrar numa igreja, olha a doutrina e o templo no !igreja lista e no !igreja doutrinas. Cada doutrina dá um bônus diferente, e ele fica mais forte a cada !igreja reformar ⛪",
		"O dízimo só sai do !trabalhar e do !daily e vai pro cofre da igreja, que não paga o Leão. Só o líder mexe no cofre, e o !igreja milagre reparte ele entre os fiéis 🙏",
		"O limite do !emprestimo cresce com quanto sua profissão paga, e imóvel e instrumento de trabalho contam. Paga antes de vencer: depois disso o banco penhora seu saldo de tempos em tempos até quitar 🏦",
		"Dívida vencida no !agiota é perigosa: os juros de atraso são maiores e o credor pode te !matar, !prender, !sequestrar ou !processar mesmo sem ter profissão pra isso 💀",
		"No !leilao, quem é coberto recebe o lance de volta na hora, e lance nos últimos minutos prorroga o fechamento, então não adianta esperar o último segundo 🔨",
		"O !bolao não tem taxa da casa: o pote todo vai pra quem acertou, proporcional à aposta, e se ninguém acertar todo mundo recebe de volta 🤝",
		"A !loteria é a única aposta em que o pote é maior do que tudo que foi pago, porque a casa põe um bônus em cima. Cada bilhete é uma chance a mais, e se só você comprou, seus bilhetes passam pro próximo sorteio 🎟️",
		"Nas !acoes o preço tende a voltar pro valor normal da empresa e toda ordem paga corretagem. DOGE3 balança muito mais que BOTE3, então escolhe pelo tanto de risco que você aguenta 📈",
		"Um título do !titulo pode ser tomado por quem pagar mais, e o dono antigo recebe parte de volta. E só conta o que tá equipado com !titulo usar 👑",
		"Se o !roubar der errado, você paga indenização pra vítima, tirada do *seu* saldo. Quanto mais grana na mão, mais caro sai o fracasso. Na !quadrilha cada um paga a sua parte, e quem tá liso demais pra pagar ainda pode ir pra cadeia 🦹",
		fmt.Sprintf("*Alarme* na mochila pode fazer um !roubar falhar na hora, e o *Cadeado* protege a mochila do !furtar. Os dois quebram quando seguram o ladrão, e cada um carrega no máximo %d alarmes. O alarme não fica sempre na !loja: ele só aparece em alguns dias, nas ofertas do Só hoje, e não sai de caixa, baú nem do !trabalhar. Contra !quadrilha o alarme dispara menos, porque perde força a cada pessoa a mais no bando, e quem acabou de ganhar bolada na aposta tá tão distraído contando dinheiro que o alarme também falha mais 🚨", alarmCarry),
		"Um segurança particular no !empregados e a bênção *Proteção* dificultam te !roubar, !furtar, !sequestrar e !matar, e seguram !quadrilha também. O segurança cobra salário todo dia junto do imposto (sem saldo, a tensão sobe, e com 3 salários atrasados ele entra em greve), ocupa uma vaga da casa e só vale um. A maldição *Olho gordo* faz o contrário 🛡️",
		"Rico perde mais moedas num roubo, mas uma fatia bem menor do saldo. Olha o *Valor de roubo* no !saldo ou no !tu pra saber o máximo que um !roubar leva de você 💰",
		fmt.Sprintf("Quem acabou de ser roubado(a) fica %s sem poder ser roubado(a) de novo, nem no !roubar nem na !quadrilha. Ninguém mais limpa a carteira de alguém num roubo atrás do outro 🛡️", core.FormatWait(robShield)),
		fmt.Sprintf("Ficha limpa protege: cada %s sem cometer crime soma %d%% de proteção contra !roubar, !furtar, !sequestrar e !matar, até %d%%. Qualquer crime zera a contagem, e ir preso não apaga ela. Olha a sua no !saldo 🕊️", core.FormatWait(cleanEvery), percent(cleanStep), percent(cleanMax)),
		"A polícia só consegue te !prender ou !denunciar se você cometeu crime há pouco tempo, e cada crime rende uma prisão só. Se tentarem te prender com a ficha limpa, você ganha uma queixa e pode !processar quem tentou ⚖️",
		"O !fugir é crime: se der certo, a ficha suja volta e a polícia pode te prender de novo, e se der errado a pena aumenta. Em compensação, cada nova prisão corta boa parte da espera pro próximo !fugir 🏃",
		"Advogado e juiz pegam bem menos cadeia, e piloto de fuga, mágico e chaveiro fogem fácil no !fugir. A bênção *Libertação* ajuda na fuga e corta a pena, e a maldição *Encosto* faz o inverso 🔓",
		"Ir preso pelo !prender ou pelo !denunciar faz a polícia levar *todo item ilegal* da mochila, o Colete à prova de balas inclusive, e arrancar os implantes clandestinos. O *Coração do Corpo-Seco* segura os implantes, e quem tem contrato com uma organização do !empresa crime pode pagar pra ela sumir com as provas e devolver a muamba 🚓",
		"A !fianca e o !hospital custam uma porcentagem do saldo de quem tá preso ou machucado, e a fiança fica mais barata conforme a pena passa. Um amigo pode pagar com !fianca @pessoa ou !hospital @pessoa 🏥",
		"O *Colete à prova de balas* te salva de um !matar que ia dar certo (só não segura a Foice nem deus). E atacar policial é furada: no !matar e no !sequestrar, a chance de ser pego na hora é maior 🦺",
		"Recompensa só se cobra no !matar. O *procurado do dia* sai entre quem usou !daily, !trabalhar ou !roubar nos últimos dias, e quanto mais rico, maior a chance de ser você. Com o *Quepe da Volante*, qualquer um caça o procurado sem virar criminoso, e o prêmio vem dobrado 🎯",
		"Foi sequestrado? Se o sequestrador for preso ou morrer, você é solto na hora. Mas depois que ele fecha o !venderrefem a polícia não te solta mais, então corre atrás de um policial pra !prender ele antes 🚔",
		"No !venderrefem, refém que ganha bem na profissão e refém famoso valem mais. Só não deixa pra última hora, que com pouco tempo de cativeiro não dá pra achar comprador 💼",
		"Emprestou no !agiota e o prazo venceu? Você pode !prender e até !matar o caloteiro sem ser da polícia nem do crime, e no !matar é cobrança legal, sem sujar a ficha 💸",
		"A maldição do !zoar corta o !trabalhar pela metade. Pra tirar, paga um !zoar limpar ou usa *Sal grosso* (!usar sal grosso), que também apaga as maldições do !amaldicoar 🧂",
		"Lançar a mesma bênção ou maldição de novo só renova o prazo, não acumula. E o *Espelho da Iara* na mochila devolve a praga pra quem jogou 🪞",
		"*Dízimo* e *Multiplicação* agem no próximo ganho e somem depois. Se te jogaram Dízimo, qualquer ganho pequeno já queima a praga. Ganhou Multiplicação? Guarda pra um ganho grande (tem teto) ✨",
		"No !processar a profissão pesa no júri: advogado e promotor puxam votos pro próprio lado, e quem é do crime perde votos. A bênção *Justiça divina* soma votos pro seu lado, e a *Excomunhão* tira ⚖️",
		"A segurança presidencial atrapalha !sequestrar, !matar e !golpe, mas não segura !roubar nem !furtar. Presidente com medo de golpe contrata um segurança no !empregados e liga o toque de recolher no !decreto, que derruba a chance do golpe e de todo crime 🏛️",
		"No !golpe, quem abre a conspiração vira presidente, e todo golpista fica com a ficha suja mesmo se der certo. Cada !golpe defender derruba a chance, e se o golpe falhar quem não escapar vai preso 🪖",
		"O !impeachment só passa com um mínimo de votos a favor *e* mais votos a favor do que contra, então mandar !impeachment nao faz diferença. Dá pra mudar o voto até fechar 🗳️",
		"Tá preso? Ganhar o !votar te tira da cadeia na posse. E presidente tem imunidade parcial: muitas vezes o !prender e o !denunciar nele não pegam, mas a ficha continua suja e dá pra tentar de novo 🏛️",
		"Teve uma ideia pra economia ou pro bot? Manda !feedback <ideia> que o dono vê. Pra espiar o que a galera já sugeriu, é !feedback ver 💬",
		"Ganhou uma bolada? Cuidado, que a notícia corre e aparece todo tipo de gente querendo um pedaço: assaltante, golpista do Pix e até primo que você nunca viu 💰",
		fmt.Sprintf("Lucrou %s ou mais numa aposta e ficou no lucro? Por %s você vira *alvo fácil*: o !roubar e a !quadrilha contra você têm %d%% a mais de chance de dar certo, o alarme perde %d%% de chance de disparar, o ladrão espera %d%% menos pra tentar de novo e leva %d%% a mais. Ser roubado(a) não tira a marca, só deixa o escudo mais curto. Quando ela acaba, quem lucrou %s ou mais em aposta nos últimos %d dias ainda fica com %d%% a mais de chance de ser roubado(a). E se continuar apostando com a marca, sobe de nível (até o %d) e fica cada vez mais na mira. O ladrão só leva o que tá no saldo, então guarda o prêmio no !poupar ou com !imovel depositar 🎯",
			Coins(jackpotMin), core.FormatWait(jackpotWindow), percent(jackpotRobBonus), percent(jackpotAlarmCut), percent(jackpotWaitCut), percent(jackpotLootBonus), Coins(luckyMin), luckyDays, percent(luckyRobBonus), markLevelMax),
		fmt.Sprintf("Vive ganhando na aposta? Contrata uma empresa no !empresa: em cada roubo ela tem chance de tirar sua fama de alvo fácil e ainda segura parte do prejuízo nos eventos de quem ganhou muito. Quem ganha %s ou mais de uma vez recebe a oferta de escolta na hora, e quem é roubado(a) às vezes recebe a oferta de ir atrás do ladrão, é só !aceitar 🕶️",
			Coins(boostPrizeMin)),
		"Vive de crime? Tem organização pra isso também: manda !empresa crime. Ela soma chance no !roubar (as maiores também no !assalto e na !quadrilha) e, se você rodar, oferece te tirar da cadeia por um preço fixo e devolver os itens ilegais que a polícia levou. Manda !aceitar fuga, !aceitar muamba ou só !aceitar pras duas 🕴️",
		"O presidente tem !decreto anistia (solta todo mundo e limpa a ficha, uma vez por mandato) e !decreto indulto @pessoa (tira alguém da cadeia na hora). Vale a pena fazer amizade com ele 🤝",
		fmt.Sprintf("O !decreto ditadura não tem volta: o presidente pode cobrar imposto muito acima do teto e o !impeachment deixa de existir. Em troca, o !golpe precisa de só %d golpistas e pode ser tentado a cada %s, e ditador que morre perde o cargo 🪖",
			tyrantCoupMin, core.FormatWait(tyrantCoupEvery)),
		fmt.Sprintf("Na ditadura, bondade do ditador é só pra ele: se ele baixa imposto, corta espera ou sobe salário por !decreto, vale inteiro pro bolso dele e o povo só sente %d%% da mudança. Já decreto ruim pega todo mundo cheio 🪖", tyrantPeopleShare),
		fmt.Sprintf("Na ditadura, todo imposto cobrado acima do teto normal deixa o povo mais revoltado, e a revolta tira até %d%% da segurança do ditador contra !matar, !sequestrar e !golpe. Quanto mais ele abusa, mais fácil fica derrubar 🔥",
			percent(revoltMax)),
		fmt.Sprintf("Ditador com medo de cair contrata batalhões no !decreto exercito: cada um custa %d%% do saldo dele, soma %d%% de segurança e deserta em %s. Pra derrubar, junta a galera antes que ele se blinde 🪖",
			battalionPercent, percent(battalionGuard), core.FormatWait(battalionTime)),
		fmt.Sprintf("Semente é dinheiro parado: o !plantar bota ela num dos seus %d canteiros, ela cresce sozinha enquanto você faz outra coisa, e o !colher traz várias unidades de uma vez. O !horta mostra quanto falta pra cada uma 🌱", gardenBeds),
		"As sementes mais caras (cacau, jabuticaba e pau-brasil) não tão na !loja: só aparecem de quebra nas extrações, e o !catar é onde mais cai semente. Antes de vender uma, lembra que plantada ela rende bem mais 🌳",
		"Planta rápida, tipo alface e milho, rende mais por hora pra quem vive no grupo. Pra deixar crescendo de noite, vai de melancia, abóbora ou mandioca, que demoram mais e enchem a mochila 🍉",
		"Fazendeiro tem mais canteiros na !horta e a planta cresce mais rápido pra ele. O agrônomo também acelera a horta e ainda colhe com qualidade melhor. Manda !profissao ver fazendeiro pra ver os números 🚜",
		"A Muda de roseira e a Semente de girassol dão um monte de rosa e girassol de uma vez, bom pra quem vive mandando presente com !dar 🌹",
		"Na cadeia não dá pra plantar nem colher, mas a horta continua crescendo. Saiu, é só mandar !colher 🌾",
		fmt.Sprintf("Horta lotada? O !horta expandir compra mais um canteiro: o primeiro sai por %s e o último por %s, até %d a mais. Pra quem planta todo dia, se paga rapidinho 🧑‍🌾", Coins(bedPrices[0]), Coins(bedPrices[len(bedPrices)-1]), len(bedPrices)),
	}
}
