---
name: economia-base
description: "Base da economia do bot (carteiras, chave canônica b.Account, profissões em profissoes.go/profissao.go, ProfessionOf/standing, deus.go, oficio.go e o !trabalhar, ralos de dinheiro, walletStore.spend, !economia resetar intervalos, vínculo de economia entre grupos com !economia vincular, ledgerDB e ledgers em vinculos.go, missões diárias do !missoes em missoes.go). Carregue antes de mexer em qualquer comando ou arquivo da economia, junto da skill economia-* do assunto."
user-invocable: false
---

# Economia: base

Regras gerais da economia, que valem pra qualquer comando dela. Os assuntos específicos ficam nas outras skills `economia-*`.

- As tabelas da economia (`carteiras`, `inventarios`, `loteria_bilhetes`, `boloes`...) guardam
  quem é a pessoa em `pessoa`/`jid`, com a chave canônica do `b.Account()`, que já junta LID e
  número.
- As profissões também vivem em código (`professions` em `profissoes.go`, separadas por área) e
  só o `name` vai pra coluna `profissao` da `carteiras`. Nunca renomeie um `name` existente: quem
  tinha a profissão volta pra `bico`. Os efeitos (campos de `Profession` em `profissao.go`) são
  aplicados em cada comando que eles afetam (`!roubar`, `!furtar`, `!pescar`, `!vender`,
  `!comprar`, `!poupar`, `!daily`, `!apostar`/`!cassino`, `!acoes`, `!processar`,
  `!prender`, `!fugir`, `!libertar`, `!reviver`, `!denunciar`, `!sequestrar`, `!venderrefem`, `!assalto`, `!usar`, `!plantar`, `!colher`, pensão e indenização), sempre lendo a
  profissão com `ProfessionOf`; efeito novo precisa de campo novo, de texto em `perks()` e do
  gancho no comando.
- O `boxLuck` da profissão (hoje só o youtuber de unboxing) dá sorteios extras no `loot` das caixas
  e baús abertos no `!usar`: o `item.roll` sorteia `boxLuck + 1` vezes e fica com a raridade mais
  alta (`lootRarity`). A chance de relíquia (`relicOdds`) e os prêmios em moeda não mudam.
- O azar da caixa (`caixaazarada.go`, tabela `apostas_recentes`) é o contrário do `boxLuck`: o
  `NoteBet` grava cada resultado de aposta com a hora, e quem tem lucro líquido de pelo menos
  `boxJinxMin` nas últimas `boxJinxWindow` (4h) leva `boxJinxDraws` a menos de sorte em toda caixa
  e baú do `!usar` (`boxJinx`, lido em paralelo com a profissão no `boxLuck`, e o saldo dos dois
  pode ficar negativo). Com sorte negativa, o `drawn` fica com o pior dos sorteios: a raridade do
  `loot`, o prêmio em moeda (`prizes`) e a chance de relíquia (`jinxedHit`, que exige acertar todos).
  A sorte positiva continua sem mexer em moeda nem relíquia. O `!bicho` não passa pelo `NoteBet` e
  fica de fora. É escondido como a série: nada no `Help`, nas dicas, nos `perks()` nem no
  trailer `Novidade:` fala disso.
- A ressaca (`ressaca.go`) lê o mesmo lucro das últimas `boxJinxWindow` (`recentWinnings`, em
  paralelo no `ProfessionOf`) e põe no `hangover` da profissão (`hungover`, o último passo da
  cadeia, que pula o deus) o `hangoverOf`: `hangoverStep` a partir de `hangoverFloor`, mais
  `hangoverStep` a cada 10x, até `hangoverMax`. É uma fração da chance, não pontos percentuais: o
  `sober` multiplica a chance de sucesso por `1 - hangover` (crimes, `!fugir`, `!libertar`,
  `!reviver`, `!denunciar`, maldição, achado e dobro do `!trabalhar`), o `shaky` multiplica o
  `bust` do `!trabalhar` por `1 + hangover` (até 95%) e o `forage` soma `hangover * hangoverBarren`
  no `barrenHauls` das extrações. Nos crimes em grupo (`!assalto`, `!golpe`, `!quadrilha`), o
  `crewHangover` fica com a ressaca do pior membro. Chance nova que deva sofrer com a ressaca
  passa pelo `sober`. Escondida como a série.
- Turnos acumulados (`turnos.go`): o `!trabalhar` e as extrações pagam a espera que ficou parada.
  O `backlog` divide o tempo desde a última vez (o `stampedAt` lê a coluna de espera, `trabalho_em`
  ou a da atividade) pela espera e limita a `backlogSpan` (1h), arredondando pra cima, então quem
  manda de vez em quando ganha o mesmo por hora que quem manda sem parar, e quem manda sem parar não
  ganha nada a mais. No `!trabalhar`, o `presidency.toil` sorteia um `work` por turno (cada um passa
  pelo `payFor`) e junta tudo num `shift` só com o `merge` (ganhos e multas somados, `turns`, `busts`,
  `doubles` e os achados em `finds`); o ferimento, o evento e a cobrança de imposto, dízimo e gravidade
  saem uma vez só, sobre o total. Nas extrações, o `walletStore.gather` roda um `dig` por tentativa
  com a mochila em memória: cada tentativa sorteia quebra (e desgaste do `grace`) e gasta os
  consumíveis do `spent`, e o `equip` é refeito depois de cada uma (`withdrawn` tira da mochila o que
  acabou), então a ferramenta quebrada e o consumível que acabou deixam de valer nas tentativas
  seguintes, e sem ferramenta as tentativas param. A primeira tentativa vira o `haul` da resposta e
  as outras ficam no `backlog` (resumidas no `backlogReport`); o ferimento é um só por comando e o
  pedágio da cratera cobra sobre tudo (`gathering.finds`). As missões `trabalho` e `extracao` andam
  pelo número de turnos. Comando de renda novo com espera pode usar o mesmo `backlog`.
- Carreira (`carreira.go`, tabela `carreiras` com `chat`, `pessoa`, `profissao` e `minutos`): cada
  profissão guarda o tempo de trampo da pessoa nela, contado em tempo de espera (turnos vezes a espera
  do `!trabalhar`), então profissão de espera curta e de espera longa sobem no mesmo ritmo por hora. O
  `ProfessionOf` lê todas as carreiras da pessoa em paralelo (`walletStore.careers`) e aplica o
  `seasoned` antes de tudo: o nível (`career.level`, o nível N precisa de `careerFirst` vezes N², até
  `careerTop`) soma `careerStep`% no `min` e no `max`, então o `shiftValue` dos eventos cresce junto.
  Trocar de profissão começa outra carreira do zero e a antiga fica guardada. O `!trabalhar` soma o
  tempo no fim (`promote` → `walletStore.build`, um `UPSERT` fora da transação do pagamento) e o
  `career.progress` põe na resposta a experiência ganha e quanto falta pro próximo nível, ou avisa
  quando sobe; o `perks()` mostra o nível e quanto falta (`career.perk`), e o menu do
  `!profissao` mostra o nível atual (`career.tag`). Deus (`divine`) não tem carreira.
  O que é da profissão também treina, bem pouco: `careerDrill` (1 minuto) por ação, contra os 8 a
  40 minutos de um turno do `!trabalhar`. A regra olha sempre a profissão pura (`professionNamed`,
  sem item, implante, droga nem presidência), pra arma ou implante não virar carreira. O `practice`
  soma quando o `fits` bate: as extrações chamam ele por tentativa com o `crafts` (o `craft` da
  profissão é daquela atividade, ou ela tem `gatherFaster`) e o `!colher` uma vez com o `farms`. O
  resto passa pelo `drillCareer` (`AddAfterCommand`), com um mapa de comando pra coluna de espera e
  pra regra da profissão (`!roubar` e `!quadrilha` pra quem tem bônus de roubo, `!prender` pra quem
  tem `jailTime`, `!reviver` pra quem tem `reviveChance`, `!abencoar`/`!amaldicoar` pra quem tem
  `faith`, `!postar`/`!publi` pra quem tem `reach` ou é da arte...): a ação conta quando a coluna de
  espera foi carimbada depois da mensagem (`walletStore.drilled`, com `drillSlack` de folga), então
  comando recusado pela espera não treina. Comando novo de profissão com espera entra nesse mapa.
- Toda profissão pode achar item no `!trabalhar` (`oficio.go`): o `loot()` usa o `finds`/`findChance`
  da profissão e, se ela não tiver `finds`, cai nos `finds` da área (`areas` em `profissao.go`) com
  `areaFindChance`. Cada área tem um instrumento de trabalho no `catalog` (campo `trade`, um por
  área, vendido na `!loja` e também na lista de achados da área). Ele vale só de estar na mochila e
  só pra quem é daquela área: o `walletStore.passives` lê ele junto dos itens de luxo com `flex`, e
  o `tooled` soma `tradePayBonus` no ganho e `tradeFindBonus` no `findBonus`. Área nova precisa de
  `finds` e de um item com `trade`. O tipo `Achados do trampo` pega o que sai do `!trabalhar` e não
  bate em nenhum tipo antes dele.
- A profissão `deus` (`god` em `deus.go`) fica fora da lista `professions`, então não aparece em
  menu nem nas listas dos comandos; o `professionNamed` reconhece ela pelo nome. Só o dono vira
  deus (`summonsGod`), e entrar ou sair dela não tem a espera de `professionChangeEvery`. Ela tem
  `divine`: o `pace` zera as esperas dos poderes (`!roubar`, `!furtar`, `!sequestrar`,
  `!processar`, `!matar`, `!reviver` e `!libertar`), o `!matar` não erra e os `perks()` dela vêm de
  `divinePerks`. O `voteWeight` dela faz o voto contar como 3 na eleição do `!votar` (guardado na
  cédula, em `weights`) e no júri do `!processar` (lido na sentença). Espera de poder nova passa pelo `pace` da profissão de quem usa.
  O `pace` também aplica o `haste` da profissão (a bênção Pressa corta e a maldição Lerdeza
  aumenta, em porcentagem), e as esperas que não passam por ele (extrações, `!prender`, `!denunciar`)
  chamam o `hasted` direto; o `!trabalhar` muda pelo `every`, que os dois encantos mexem no `apply`.
  Só ela usa o `!milagre <valor>` (`milagre.go`), que credita moedas novas (nada sai do saldo do
  deus) pra cada membro atual do grupo, menos o próprio deus e o bot, até `miracleMax` por pessoa;
  com gente marcada, só quem foi marcado recebe (`chosenMiracle`, sem o próprio deus).
  O `!milagre <item> [quantidade]` faz o mesmo com um item do `catalog` (`bestowItem`), até
  `creationMax` por pessoa; sem nome de item na mensagem, o número vira moeda.
  O `!castigo <valor>` (`castigo.go`) é o contrário: tira o valor (número, porcentagem, `metade`
  ou `tudo`, calculado e limitado pelo saldo de cada um no `smite`) de todo membro do grupo, ou só
  de quem foi marcado, e as moedas somem. O `!criar <item> [quantidade]` (`criar.go`) põe
  qualquer item do `catalog` na mochila do deus (até `creationMax` de uma vez), inclusive os de
  luxo e os ilegais, sem cobrar nada. Com o submodo `aovivo` ligado, o `!aovivo` solta um evento
  ao vivo na hora (skill `economia-aovivo`).
- O `!economia resetar intervalos` (dono, em `economia.go`) zera as colunas `*_em` de espera da
  `carteiras` (`resetCooldowns`), da conversa toda ou só de quem foi marcado. Ele também zera
  `preso_ate`, `morto_ate`, `crime_em`, `escudo_ate` e `ficha_em` e apaga a linha de `sequestros`
  de quem é refém, então solta da cadeia, da morte e do cativeiro, tira o escudo de roubo e limpa
  a ficha (com o `ficha_em` zerado a pessoa volta pra ficha limpa cheia). Também enche a saúde (`saude` e
  `saude_em`). Nunca mexe em saldo, mochila,
  poupança, empréstimo nem presidência. Coluna de espera nova entra na lista do `resetCooldowns`.
- Os ralos de dinheiro existem porque a economia só tinha torneira: o dinheiro gasto neles some
  (nunca vai pra ninguém), a não ser quando o texto diz o contrário. Preço de ralo, quando dá, é
  uma porcentagem do saldo com piso, pra pesar igual pra rico e pobre. Pra cobrar e fazer alguma
  coisa na mesma transação, use `walletStore.spend` (debita com `debitBalance` e chama o resto).
- O `ProfessionOf` lê tudo de uma vez (`walletStore.standing`) e aplica, depois do
  `presidential()`, o imóvel (`housed`), o segurança particular do `!empregados` (`escorted`,
  com o `watch` do melhor segurança contratado, lido pelo `staffKinds` no `standing`), a ficha
  limpa (`cleared`, com o `ficha_em` lido no `standing`, skill `economia-crime`) e a maldição do `!zoar` (`hexed`). Os implantes do `!implante` (`implanted`,
  lidos em paralelo pelo `walletStore.body`) entram logo depois dos itens com `flex` (`adorned`).
  Os seguidores do `!postar` (`famed`, lidos em paralelo pelo `walletStore.audience`, skill
  `economia-fama`) entram antes dos encantos, então toda profissão lida pelo `ProfessionOf` sabe
  o nível de fama da pessoa (`celebrity()`).
  Efeito novo que mude `guard`, ganho ou bônus de alguém por fora da profissão entra nesse mesmo
  ponto.
- Grupos podem dividir uma economia só (`vinculos.go`, tabela `economia_vinculos` com `chat` e
  `economia`). O `!economia vincular <nome do grupo>` (dono, achado pelo `b.GroupNamed` do `core`)
  faz a conversa atual usar a economia do outro grupo, e quem já usava a economia dela vai junto.
  O `!economia desvincular` volta pra economia própria e o `!economia vinculos` lista os grupos. O
  vínculo aponta sempre pro grupo principal, que nunca tá vinculado a ninguém, então a troca é
  idempotente. O que o grupo tinha antes de vincular fica congelado com o `chat` dele.
- A troca acontece no banco, num lugar só: o `walletStore.db` é um `ledgerDB` e as transações são
  `ledgerTx`, que trocam todo argumento que for JID de grupo (string `...@g.us` ou `types.JID`)
  pelo grupo principal da economia (`ledgers.of`, um cache carregado no `New`). Por isso consulta
  nova da economia só precisa passar pelo `s.db` ou pelo `tx` de sempre, e nunca deve abrir o
  `b.Messages.DB` direto. Chave que não é SQL e precisa ser igual em toda a economia usa o
  `ledgers.of(chat)` (as sementes do `dailyRand` e do `auctionRotation`, que decidem loja, luxo,
  mercado negro e leilão do dia). O estado em memória dos jogos (`sync.Map` de eleição, tribunal,
  crash...) continua por grupo, e o `!milagre`/`!castigo` sem marcação pegam só quem tá no grupo
  atual.
- As missões diárias (`missoes.go`, tabela `missoes`) são `missionsPerDay` metas por pessoa,
  sorteadas de `missionGoals` com o `dailyRand` (semente da economia + `acc.Key`), então mudam à
  meia-noite e não ficam guardadas: o banco só guarda o progresso de cada uma (`progresso`/`meta`)
  no `dia` de hoje, e apaga os dias velhos da pessoa na próxima escrita. Cada comando que conta pra
  uma missão chama `b.advance(chat, acc, goal, quantidade)` depois que a ação deu certo
  (`!trabalhar`, extrações, ganho no `!apostar`/`!cassino`, `!roubar`/`!furtar` bem-sucedido,
  venda pro mercado no `!vender` e compra no `!comprar` da loja). O `advance` só enfileira em
  memória (`missionLog`) se a meta for de hoje daquela pessoa; o `settleMissions`, registrado no
  `AddAfterMessage` antes do `announcePayouts`, grava o progresso, paga a recompensa
  (`target / step * pay`) na hora em que a missão fecha e avisa no grupo depois da resposta do
  comando. Quem fecha todas leva `missionBonus` e uma `missionPrizeItem`. A recompensa entra direto
  no saldo, sem imposto nem dízimo. Missão nova precisa de `id` em `missionGoals` e do `advance` no
  comando, e as metas em moeda usam `step` de 100 pra recompensa ficar proporcional.
- As rotinas da economia (`runTaxes`, `runLottery`, `runAnimalGame`, `runAuctions`, `runLoanSeizures`, `runWanted`, `runGravity`, `runSuccession`) pulam as
  linhas de grupo congelado (`ledgers.linked`) e anunciam com o `announce`: manda em todo grupo da
  economia que tá com o submodo `anuncios` (`core.ModeAnnouncements`, filho da `economia`) ligado e
  onde quem aparece no aviso passa no `AllowedIn` (o `announce` aceita mais de uma pessoa em destaque, como os ganhadores do `!bicho`: todas são marcadas e passam juntas pelo `AllowedIn`). O submodo é por grupo, como todo modo: grupos
  vinculados dividem as carteiras, mas cada um liga o `anuncios` dele, e sem ele a rotina roda do
  mesmo jeito (cobra, sorteia, paga), só não avisa. Aviso automático novo da economia passa pelo
  `announce`, nunca pelo `SendText` direto. A exceção é o aviso de atualização do bot, que também
  sai pros grupos com o `anuncios` ligado, mas fica no `core` (`versao.go`, seção "Versões e
  novidades" do CLAUDE.md) e não passa pelo `announce`.
- A sugestão da espera (`enquantoisso.go`): quando o `!trabalhar`, uma extração ou o `!extracao`
  inteiro é recusado pela espera, o `meanwhile` põe na resposta um bloco "👉 Enquanto isso" com o
  que a pessoa pode fazer agora: as rendas já liberadas (o `!trabalhar` pelo `workCooldown` da
  presidência e as extrações pelo `paced` + `equip`, a mesma conta do `extract`, então só aparece
  extração com ferramenta e na espera do comando sozinho, não na do atalho), a horta (canteiro
  pronto pro `!colher`, ou vazio com semente na mochila pro `!plantar`) e os anúncios dos outros na
  `!feira`. Sem nada disso, sorteia uma linha fixa (blackjack, crash, corrida, missões, feira). O
  comando que foi recusado fica de fora (`busy`). Renda nova com espera que deva aparecer ali entra
  no `idleBoard`.
- As dicas (`dicas.go`) saem pelo `offerTip`, registrado no `AddAfterMessage` depois do
  `announcePayouts`: numa mensagem qualquer, se já passou `tipEvery` desde a última dica daquela
  conversa, o `tipBoard` tira a próxima do baralho dela (embaralhado, sem repetir até acabar) e o
  bot manda, só se o submodo `dicas` (`core.ModeTips`, filho da `economia`) tiver ligado. O
  relógio e o baralho ficam em memória: depois de reiniciar, a primeira dica de cada conversa sai
  `tipEvery` depois da primeira mensagem. O `!dica [assunto]` manda uma na hora, filtrando pelo
  texto. O baralho junta as dicas escritas à mão (`economyTips`) com uma por comando da economia,
  montada do `Help` dele (`commandTips`), então comando novo já vira dica sozinho. Ao mudar uma
  mecânica citada no `economyTips`, corrija a dica junto.
- O `!ricos` (`economia.go`) mostra, embaixo do ranking, quem caiu desde o último `!ricos` e
  ficou mais pobre (`quedas.go`). O ranking de cada `!ricos` fica salvo na `ricos_ranking`, e os
  prejuízos vão somando na `prejuizos` por pessoa e motivo (`noteLoss`): aposta (o `tallyBet`, o
  `!roleta` pelo `NoteBet` e o `!bicho` na extração), roubo (`!roubar` e `!quadrilha`) e
  imposto (`runTaxes`). O motivo que aparece é o de maior valor; sem nenhum anotado, vira "torrando
  dinheiro". Depois de mostrar, as duas tabelas da economia são zeradas e o ranking novo é gravado.
  Prejuízo de outro tipo que deva aparecer no `!ricos` precisa de um `lossCause`, do texto no
  `richFall.text` e do `noteLoss` no caminho que tira o dinheiro.
- Toda aposta segue o `betRule` (`stakeRule` em `limiteaposta.go`): o `most` é o limite de uma aposta
  só, `stakeSharePercent` do saldo e nunca mais do que falta pra pessoa chegar em `betWindowCap`
  apostados nas últimas `betWindow` (2h); o `whole` é o que o `tudo`/`all` aposta, `allInSharePercent`
  do saldo (também até o `most`). O volume vem da tabela `apostas_volume` (migração `00121`): o
  `noteWager` grava cada valor apostado, com a hora, na mesma transação que tira a aposta do saldo
  (`SettleBet`, que o `holdStake` também usa, `betOnPool` do `!bolao` e `placeAnimalBet` do
  `!bicho`), e apaga as linhas da pessoa que já saíram da janela. O `betRuleOf` soma a janela
  (`wagered`, que também diz em quanto tempo a aposta mais velha sai, o `freed`) e monta a regra. O
  `most` também passa pelo stop-loss: o `betRuleOf` lê em paralelo os resultados das últimas
  `stopLossWindow` (2h, dentro das 4h que a `apostas_recentes` do `NoteBet` guarda) com o `betOutcomes`, e o
  `room` deixa apostar só o que falta pro prejuízo líquido da janela (`lost`, que fica negativo com
  lucro) chegar em `stopLossPercent` (70%) do que a pessoa tinha (saldo + `lost`). Bateu, o `most` vira
  zero e o `reopens` diz quando a perda que segura sai da janela, supondo o saldo de agora. Saldo novo
  (trabalho, daily) aumenta o que tinha e libera de novo. O stop-loss é visível, ao contrário do
  `stakeRisk`: o `tooMuch` diz quanto a pessoa perdeu e quanto ainda dá. O `!bicho` respeita o
  stop-loss, mas não passa pelo `NoteBet`, então a perda dele não conta. O `affordable` recebe a regra: as apostas (`placeStake`, que o `PlaceBet` usa, e `!bicho`) passam o
  `betRuleOf`, e o `!lance` do leilão passa o `bidRule`, em que os dois são o saldo inteiro e não tem
  janela. Com o limite da janela batido (`most` zero), a aposta nem é lida; valor acima do `most` mas
  dentro do saldo é recusado. Os dois textos saem do `tooMuch`, que diz qual limite pegou. O `affordable`
  devolve o `whole`, e toda aposta a partir dele passa pelo `allIn` (skill
  `economia-eventos`). `metade` e porcentagem continuam calculados sobre o saldo todo. Jogo de aposta novo passa pelo `PlaceBet`/`placeStake` e grava
  pelo `noteWager` onde tira o dinheiro.
- Os jogos de cassino (`casinoGames`) sempre deixam uma vantagem pequena pra casa: o `!apostar`
  ganha só `coinWinChance` das vezes, e o `!cassino` transforma `slotHouseCut` das vitórias em
  derrota (gira de novo até sair sem par), sem mudar a tabela de prêmios do `Help`. Nos dois, quanto
  maior a aposta de uma vez, maior a chance de perder: o `stakeRisk` (`risco.go`) soma
  `stakeRiskStep` a cada 10x acima de `stakeRiskFloor`, até `stakeRiskMax`. Os jogos que passam
  pelo `PlaceBet` leem essa chance pelo `PrizeBoost.Risk`, que soma a ela o calor da série (veja
  abaixo, limitado a `streakHeatMax`) e `decreeRiskStep` por ponto do decreto `cassino` (skill
  `economia-presidencia`). O `PlaceBet` devolve um `wager` com o
  `Boost` da hora da aposta (`PrizeBoost`), guardado na aposta e no save das rodadas (`Decree`, `Luck`, `Heat`,
  `Solace`, `Rescued`): o `decree` é o nível positivo do decreto `cassino` (lido da presidência), que o
  `Pay` soma no lucro e o `Tag` mostra (📜), mas que também soma `decreeRiskStep` de chance de perder
  por ponto no `edge` (+10% de lucro, +4,5% de chance de perder no máximo), e nada no `Help`, no `Tag`
  ou no menu do `!decreto` fala desse risco; o `levy` é a taxa do decreto negativo, cobrada em cima do
  valor apostado pelo `SettleBet` (e pelo `holdStake`, que passa por ele) e mostrada pelo `BetLevyText`
  (skill `economia-presidencia`); o `luck` é o `casinoLuck` da profissão (Baralho do Zé Pelintra e afins), que
  o `Pay` soma no lucro e o `Tag` mostra (🃏), e o `rescued` é sorteado ali com o `rescueOdds`, e na
  derrota o `Rescue` devolve a aposta inteira (no `!crash` e na `!corrida`, pelo `payRescues` no fim
  da rodada). Os jogos que seguram a aposta durante a rodada descontam pelo `holdStake` e pagam pelo
  `payPrize`. No `!apostar` e no `!cassino` ela entra na chance de `jinxed` do `gamble` (a mesma do
  Azar, mas sem passar pelo `jinxProof`). No `!blackjack` a
  mesma chance (com a aposta já dobrada, se dobrou) entra no `dealerDraw`: quando a carta do topo
  ia fazer a banca estourar, ela é trocada por uma do baralho que não estoura. Nos jogos de
  resultado compartilhado o vício muda a rodada de todo mundo: no `!crash`, o `rig` da decolagem
  sorteia pela aposta de maior `Risk` e, se cair, fica com o menor de dois `drawCrashPoint`; na `!corrida`, o
  `rig` da largada tira o cavalo vencedor quando ele é de quem caiu no sorteio e refaz os `frames`;
  na `!roleta` (pacote `games`), o `pull` recebe o `Risk` da aposta e dispara na câmara da
  vez; e no `!bicho` o `riggedAnimalDraw` sorteia de novo o milhar só com o `stakeRisk`, sem a
  série (skill `economia-dinheiro`).
  Também é escondido,
  como a série: nada no `Help`, nas dicas, no `Tag`, nos `perks()`, nas respostas da IA nem no
  trailer `Novidade:` dos commits fala disso. Por cima disso
  vem a série de apostas (`serie.go`, colunas `aposta_serie` e `aposta_lucro` da `carteiras`): o
  `NoteBet`, chamado no fim de toda aposta (pelo `tallyBet` e pelo `!roleta`), soma o resultado no
  `aposta_lucro` e conta mais uma aposta enquanto a pessoa segue no lucro. Quando uma perda leva
  o lucro a zero ou menos, os dois voltam pra zero. O `PlaceBet` lê a série (`betHeat`,
  `streakHeatStep` por aposta, até `streakHeatMax`) e põe no `heat` do `PrizeBoost`, que vira pontos
  percentuais a mais de chance de perder no `Risk`. O prêmio não é cortado: o `Pay` paga o lucro
  cheio com o bônus de sorte da profissão. Isso é escondido de propósito:
  nada no `Help`, nas dicas ou no `Tag` do resultado fala da série. Jogo de aposta novo passa pelo
  `PlaceBet` e pelo `NoteBet` pra entrar nela.
- A maré de azar (`solace` em `serie.go`, coluna `aposta_perda` da `carteiras`) é o contrário da
  série: o `NoteBet` soma cada perda líquida no `aposta_perda` e desconta cada lucro (nunca fica
  abaixo de zero), e o `PlaceBet` lê junto com a série (`betRecord`) e põe no `solace` do
  `PrizeBoost` o `betSolace`: `solaceStep` pontos percentuais com `solaceFloor` de perda, mais
  `solaceStep` a cada 10x, até `solaceMax`. Ele não mexe no prêmio, só na chance: o
  `PrizeBoost.edge` soma `stakeRisk` e calor e desconta o `solace`. Positivo, vira o
  `Risk` de cima; negativo, vira o `Favor`, e a casa vicia a favor de quem tá perdendo, espelhando
  o `Risk` em cada jogo: o `gamble` joga de novo, sem `jinxed`, uma rodada perdida; o `dealerDraw`
  troca pelo `stackDeck` a carta da banca por uma que estoura; o `rig` do `!crash` fica com o maior
  de dois `drawCrashPoint` (só quando a aposta de maior `edge` da rodada tá a favor); o `rig` da
  `!corrida` (`swaysRace`) sorteia outro vencedor quando o cavalo de quem tá a favor perdeu; e o
  `pull` da `!roleta` muda a bala de câmara quando ela ia disparar. O `stakeRiskStep` é maior que o
  `solaceStep` de propósito: apostar de uma vez o que perdeu sempre fica com `Risk`, então a maré
  só ajuda quem volta com aposta menor que a perda. É escondido como a série, e o `!bicho` não
  passa pelo `PlaceBet` e fica de fora.
