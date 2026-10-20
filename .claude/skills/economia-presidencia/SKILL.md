---
name: economia-presidencia
description: "Presidência do bot (presidencia.go, tabela presidencias, eleição do !votar, vaga e sorteio do presidente em sucessao.go/runSuccession, poderes presidential(), imunidade presidentImmunity, segurança cordon, !decreto com percentDecrees/toggleDecrees/onceDecrees e os impostos de decreto PixLevy/levy, !impeachment, !renunciar e decretos que sobram, !golpe). Carregue antes de mexer em presidente, decreto, imposto de decreto ou eleição."
user-invocable: false
---

# Economia: presidência

Presidente, decretos, impeachment, renúncia e golpe.

- A presidência (`presidencia.go`) fica na tabela `presidencias`, uma linha por conversa (`pessoa`/`jid`,
  `mandato_ate`, as cargas de abuso e as colunas dos decretos). O `!votar @pessoa`
  só abre eleição se não tem mandato em andamento; a urna é efêmera (`elections`, uma por conversa),
  fica aberta `electionWindow` e precisa de `electionQuorum` votos. Empate no topo com quórum abre o
  segundo turno (`runoffBetween`/`openRunoff`): uma eleição nova no lugar da antiga, com os votos
  zerados, `runoff` só com os empatados (o `castBallot` recusa outro nome com `outside`), aberta
  `runoffWindow` (`closes()`), sem quórum e salva no `KeepGame` com o `Runoff`. Empate no segundo
  turno vai pra sorte. Quem ganha fica `termLength` (2h) no cargo. Os poderes entram pelo `ProfessionOf`, que aplica `presidential()` na
  profissão de quem é presidente (`min`/`max` do `!trabalhar` somados de `presidentPayBonus` por cento, `dailyBonus` somado de `presidentDailyBonus`, mais `cordon`, `releaseChance` e `reviveChance` mínimos e, se não for da polícia,
  `jailTime`/`arrestEvery` próprios), então valem sozinhos em todo comando que lê a profissão. O
  `cordon` (`presidentGuard`) é a segurança presidencial: fica fora do `guard`, então não protege
  contra `!roubar`, `!furtar` nem `!quadrilha`, e só entra onde a conta usa o `violenceGuard()`
  (`guard + cordon`: `!sequestrar`, `!matar` e a chance do `!golpe`). O presidente pode ser preso,
  mas tem imunidade parcial: o `immunity()` da profissão devolve `presidentImmunity`. No `!prender`
  e no `!denunciar` ela vai no `arrestOdds` do `walletStore.arrest`, sorteada depois da chance da
  denúncia; se segurar, o `arrest` volta com `immune`, ninguém é preso, o `crime_em` continua (dá
  pra tentar de novo) e a resposta é o `immunityText`. No flagrante na hora, o `flagrantOdds` da
  profissão multiplica a chance de ser pego por `1 - immunity()`, e a batida do `!mercadonegro`
  (`raidOdds`) também. O `noSnitch` não vale pra ele e o `inaugurate` zera o `preso_ate` de quem
  toma posse. Caminho novo que mande alguém pra cadeia precisa passar pelo `immunity()`, ou pelo
  `flagrantOdds` se for flagrante na hora. Tudo some quando o `mandato_ate`
  passa, porque toda leitura da presidência filtra por ele, menos os decretos depois de uma renúncia
  (`!renunciar`, abaixo).
- O `!decreto` (`decreto.go`) é só do presidente e lê três tabelas em código, cada decreto com os
  nomes que aceita e a coluna dele na `presidencias`: `percentDecrees` (um número guardado na
  coluna, limitado a `lo`/`hi` na leitura; negativo corta e positivo aumenta, como no `trabalho`,
  que vai de `-workDecreeMaxCut` a `+workDecreeMaxRaise` na espera), `toggleDecrees` (liga/desliga,
  o `!decreto` sem número inverte) e `onceDecrees` (até `limit` usos no mandato, contados na coluna
  pelo `spendDecree`, que roda o efeito na mesma transação e desfaz tudo com `errDecreeIdle` quando
  não tinha o que fazer, sem gastar o uso). O menu, a ajuda do `!decreto` no `economyCommands`, o
  "em vigor" e o "sobrando" do `!presidente` saem dessas tabelas, e o `inaugurate` zera todas as
  colunas (`decreeColumns`). O `ProfessionOf` pendura os decretos da conversa na profissão
  (`job.decrees`), então os ganchos leem `job.decrees.level(...)` e mostram `job.decrees.note(...)`:
  `daily`, `juros` (pontos na taxa do `!poupar`), `pena` (no `sentence`, então vale pra toda ida pra
  cadeia), `extracao` (espera do `!pescar`, `!minerar`, `!vasculhar` e `!catar`, e o intervalo dos
  `!empregados`), `fama` (espera do `!postar`, pelo `postEvery`) e o toque de recolher (`curfew()`, tirado da chance de `!roubar`, `!furtar`,
  `!sequestrar`, `!matar`, `!assalto` e `!quadrilha`). Quem não lê a profissão chama
  `b.Wallets.decrees`: o `trabalho`/`salario` (`workCooldown`/`payFor`), o `pix` (taxa de quem
  recebe no `!transferir`, que vai pro presidente, a não ser que ele esteja na transferência), o
  `iof` (porcentagem dos juros do `!poupar` cobrada no `redeem` e paga pro presidente, a não ser que
  seja ele resgatando; o `!decreto imposto poupanca` cai nele porque o `pickDecree` tenta antes as
  duas primeiras palavras juntas), a `renda` (porcentagem do que o `!trabalhar` e o `!daily` pagam,
  tirada no `work`/`claim` com o `incomeLevy` e paga pro presidente, a não ser que seja ele
  ganhando; o `!decreto imposto trabalho` e o `!decreto imposto daily` caem nela pelo mesmo motivo
  do `iof`), a `loja` (imposto em cima do `!comprar` da `!loja` e do `!petshop`, cobrado no
  `buyItem` junto do preço e pago pro presidente, a não ser que seja ele comprando; o `!decreto
  imposto compra` cai nela pelo mesmo motivo do `iof`, e a `!luxo` e o `!mercadonegro` não pagam), a
  `fianca` (imposto em cima do preço do `!fianca`, cobrado de quem paga no `buyExit` pelo `levy` do
  `exit` e pago pro presidente na mesma transação, a não ser que seja ele pagando; o `!decreto
  imposto fianca` cai nela pelo mesmo motivo do `iof`, e o `!hospital` e o `!resgate` não pagam).
  O `cassino` (`casinoDecree`, coluna `cassino_pct`, que voltou na migração `00124`) vai de
  `-betTaxDecreeMax` a `+casinoDecreeMaxRaise` (10 pros dois lados, e `-tyrantBetTaxMax` na ditadura,
  pelo `tyrantLo`). O `PlaceBet` lê a presidência em paralelo e monta o `PrizeBoost` da hora da aposta
  (skill `economia-base`): positivo vai pro `decree`, e cada ponto soma 1% no lucro de quem ganha e,
  escondido, `decreeRiskStep` (0,45 ponto percentual) na chance de perder; negativo vira o `levy`
  (`casinoLevy`), uma taxa em cima do valor apostado, ganhando ou perdendo, que o `SettleBet` cobra
  junto da aposta na mesma transação e paga pelo `payLevy` (o presidente não paga na aposta dele, e
  depois do `!renunciar` vai pro ralo). Como o `SettleBet` é quem tira a aposta (o `holdStake` também
  passa por ele), a taxa vale no `!apostar`, `!cassino`, `!roleta`, `!crash`, `!corrida` e no
  `!blackjack`, inclusive no dobro (o `levy` vai no save da mão, `levySave`). O `BetLevyText` mostra a
  taxa na resposta. O `!decreto imposto aposta 5` cai nele, e o `taxedRun` dos apelidos
  `impostoaposta`/`taxaaposta` troca o sinal. O `!bicho` não lê (é clandestino). Os de uma vez são `imposto`, `auxilio`
  (`presidentAid` de moeda nova pra cada carteira da conversa), `caixa` (uma `presidentBoxItem`, a
  caixa misteriosa comum, na mochila de cada carteira da conversa), `anistia` (zera `preso_ate` e o
  `crime_em` dentro da `crimeWindow`), `indulto @pessoa` (`presidentPardons` por mandato), `confisco
  @pessoa` (`presidentSeizePercent` do saldo pro presidente) e `acoes <ticker> <pct>` (mexe direto
  no `acoes_precos`, e a reversão do passeio aleatório puxa o preço de volta com o tempo). Decreto
  novo precisa de coluna numa migração, de entrada numa das três tabelas (e no mapa `decrees()`, se
  for de uma vez) e do gancho no comando.
- Cargo vago vira sorteio (`sucessao.go`). A rotina `runSuccession` passa a cada `core.ReminderTick`
  pelas economias que são grupo (`economies`, só o grupo principal) e com a economia ligada em algum
  grupo. Sem mandato em andamento e sem linha na `presidencia_vagas`, ela abre a vaga
  (`openVacancy`, com `sorteio_em` em `vacancyWindow`) e anuncia pelo `announce` que dá pra eleger
  pelo `!votar` até lá. Quando o `sorteio_em` passa e não tem eleição rolando em nenhum grupo da
  economia (`electionIn`), o `drawSeat` sorteia, com a mesma chance pra todo mundo, entre quem mandou
  mensagem nas últimas `drawActivity` (24h) em algum grupo da economia e ainda tá nele
  (`activeMembers`, pelo `core.MessageStore.CountsBySender` e o `MemberOwners`). Sem ninguém ativo,
  a vaga fica aberta e o sorteio tenta de novo no próximo tick. O `drawPresident` apaga a vaga e dá
  posse na mesma transação (`swearIn` com `sorteado = 1`, `votos = 0`), e o `headline` do
  `!presidente` diz que saiu no sorteio. Toda posse (`swearIn`, do `inaugurate` e do sorteio) apaga a
  vaga, e o `!renunciar` e o `!impeachment` abrem a vaga na mesma transação (`vacate`), então o
  texto deles já traz o prazo (`vacancyNote`) e a rotina não avisa de novo. O `!presidente` sem
  presidente e a eleição anulada também mostram o `vacancyNote`.
- O `!impeachment` (`impeachment.go`) tira o presidente antes do fim do mandato. O primeiro voto
  abre a votação (`impeachments`, uma por conversa, efêmera e salva com `KeepGame`), presa ao
  presidente da abertura, que fica aberta `impeachmentWindow`. Cada um vota `sim` ou `nao` (sem
  nada é sim) e pode mudar o voto até fechar, com o peso do `ballotWeight` (o voto de deus vale 3).
  Passa com pelo menos `impeachmentQuorum` votos a favor e mais a favor do que contra, e aí o
  `oust` joga o `mandato_ate` pra agora e abre a vaga, então os poderes e os decretos caem
  sozinhos e o `!votar` volta a abrir eleição. A abertura carimba a coluna `impeachment_em` da `presidencias`
  (zerada no `inaugurate`), e o próximo pedido contra o mesmo mandato só sai depois de
  `impeachmentEvery`. Se o presidente da abertura já saiu do cargo quando a votação fecha
  (renúncia ou fim do mandato), o `impeachmentResult` só avisa e não conta nada.
- Presidente que sai do grupo (ou é removido) perde o cargo (`abandono.go`). O `unseatDeserter`,
  registrado no `AddDeparture` do `core`, recebe quem saiu (o `Leave` do `*events.GroupInfo`, ou
  quem o próprio bot removeu no `!ban` e no ban das advertências, pelo `b.Departed`) e, se o presidente da
  economia daquele grupo tá entre quem saiu e não continua em nenhum outro grupo da economia
  (`memberElsewhere`, pelo `MemberOwners` de cada grupo do `economyGroups`), tira ele pelo mesmo
  `oust` do impeachment: `mandato_ate` vira agora, a vaga abre e os poderes e decretos caem. O
  aviso sai pelo `announce`, com o `vacancyNote`.
- O `!renunciar` (`renunciar.go`) é o presidente largando o cargo: o `resign` joga o `mandato_ate`
  pra agora, marca `renunciou = 1` na `presidencias` e abre a vaga, então os poderes caem na hora e
  o `!votar` volta a abrir eleição, mas os decretos continuam valendo até o próximo `inaugurate`, que zera o
  `renunciou`. Por isso o `walletStore.decrees` e o `presidency` leem a linha com
  `mandato_ate > agora OR renunciou = 1`; com o mandato vencido, o `presidency` devolve `found`
  falso e um `presidency` só com os `decrees` (sem presidente). Os decretos de imposto continuam
  cobrando, mas viram ralo: o `levy` da presidência sai com o `president` vazio, o
  `collected()` do `PixLevy` fica falso, o `payLevy` e o `transferWithPix` não
  creditam ninguém, e os textos usam o `pocket()`/`office()` ("pro ralo"). Todo imposto de decreto
  novo monta o `PixLevy` pelo `p.levy(decreto, quem paga...)` e menciona o `levy.payee()`. O
  `!presidente` sem presidente mostra esses decretos que sobraram (`leftoverText`).
- O `!golpe` (`golpe.go`) tira o presidente na marra. O primeiro `!golpe` abre a conspiração
  (`coups`, uma por conversa, efêmera e salva com `KeepGame`), presa ao presidente da abertura, e
  quem abriu é o líder (`plotters[0]`). Quem manda `!golpe` entra no golpe e quem manda
  `!golpe defender` entra na defesa (`coupLoyal`), inclusive o presidente; não dá pra trocar de
  lado. A abertura carimba a coluna `golpe_em` da `presidencias`, e o próximo golpe contra o mesmo
  mandato só sai depois de `coupEvery`. Quando a `coupWindow` acaba, o golpe precisa de `coupMin`
  golpistas (sem esse mínimo se deus tiver no golpe). A chance (`coupAttempt.odds`) é `coupChance`
  mais `coupHelp` por golpista a mais, menos `coupHelp` por defensor, mais o `crime` da arma de
  cada golpista, menos o `guard` da arma de cada defensor (`arsenal`), menos o `guard` do
  presidente e o toque de recolher; deus golpista dá 100% e deus defendendo (ou presidente) dá 0%.
  Todo golpista leva `crime_em`. Se der certo, o `seize` troca a linha da `presidencias` pro líder
  com `mandato_ate` de `termLength`, `golpista = 1`, `sorteado = 0`, `golpe_em` agora e abusos e decretos zerados
  (o `UPDATE` exige o presidente da abertura ainda no cargo, senão falha com `errCoupVacant`); o
  presidente derrubado só perde o cargo. Se der errado, os golpistas passam pelo `jailCrew` do
  `!assalto`. O `inaugurate` zera `golpe_em` e `golpista`.
- A ditadura (`ditadura.go`) é o `tyrannyDecree`, um decreto do `toggleDecrees` com `lasting`: o
  `toggleDecreeRun` recusa desligar quando ele já tá ligado, então só cai com o fim do mandato, o
  `!golpe`, a morte do ditador ou o `!renunciar` (o `resign` zera a coluna `ditadura`). Com ela
  ligada, o `decrees.limits` troca o teto dos decretos de imposto pelo `tyrantHi` (e o piso da taxa do
  `cassino` pelo `tyrantLo`), e tanto o `level` quanto a validação do `percentDecreeRun` leem os
  limites por ele. Quando a ditadura cai, o `level` volta a cortar no teto normal sozinho. O
  `!impeachment` não abre e, se a ditadura sair no meio da votação, o `impeachmentResult` anula. O
  `!golpe` lê `coupMin()`/`coupEvery()` dos decretos (`tyrantCoupMin`, `tyrantCoupEvery`), e o
  mínimo da abertura fica guardado no `need` da conspiração (e no save). A revolta (coluna
  `revolta`) sobe no `stirRevolt` a cada imposto cobrado pro ditador, chamado pelo `payLevy` e pelo
  `transferWithPix`, somando o `excess` do `PixLevy` (o quanto o nível passa do
  limite normal, pelo `abuse`, também salvo no `levySave`). O `!decreto exercito [batalhões]`
  (`armyDecree`, só na ditadura) cobra `battalionPercent` do saldo (piso `battalionFloor`) por
  batalhão, um ralo, e empurra a coluna `exercito_ate` em `battalionTime` por batalhão, até
  `armyBattalions`. A proteção cai em linha reta até o `exercito_ate` (`decrees.army`). As duas
  entram pelo `tyrannical`, aplicado no `ProfessionOf` logo depois do `presidential()` (os
  `decrees` são pendurados antes): `cordon` mais o exército menos a revolta (`revoltMax` no
  máximo), então valem no `!matar`, no `!sequestrar` e no `!golpe` pelo `violenceGuard()`. A morte
  derruba o ditador: o `bury` chama o `deposeTyrant`, que joga o `mandato_ate` pra agora e abre a
  vaga, e todo texto de morte mostra o `deposedText` quando o `death.deposed` vem marcado. As
  colunas `revolta` e `exercito_ate` entram no `decreeColumns()` pelo `tyrannyColumns`, então o
  `inaugurate` e o `seize` também zeram as duas.
