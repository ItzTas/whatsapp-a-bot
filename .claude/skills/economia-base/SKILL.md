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
- Os jogos de cassino (`casinoGames`) sempre deixam uma vantagem pequena pra casa: o `!apostar`
  ganha só `coinWinChance` das vezes, e o `!cassino` transforma `slotHouseCut` das vitórias em
  derrota (gira de novo até sair sem par), sem mudar a tabela de prêmios do `Help`. Nos dois, quanto
  maior a aposta de uma vez, maior a chance de perder: o `stakeRisk` (`risco.go`) soma
  `stakeRiskStep` a cada 10x acima de `stakeRiskFloor`, até `stakeRiskMax`. Os jogos que passam
  pelo `PlaceBet` leem essa chance pelo `PrizeBoost.Risk`, que soma a ela o calor da série (veja
  abaixo, limitado a `streakHeatMax`) e `decreeRiskStep` por ponto do decreto `cassino` positivo (skill
  `economia-presidencia`). No `!apostar` e no `!cassino` ela entra na chance de `jinxed` do `gamble` (a mesma do
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
  cheio com os bônus de decreto e sorte. Isso é escondido de propósito:
  nada no `Help`, nas dicas ou no `Tag` do resultado fala da série. Jogo de aposta novo passa pelo
  `PlaceBet` e pelo `NoteBet` pra entrar nela.
