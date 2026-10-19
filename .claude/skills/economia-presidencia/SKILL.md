---
name: economia-presidencia
description: "Presidência do bot (presidencia.go, tabela presidencias, eleição do !votar, poderes presidential(), imunidade presidentImmunity, segurança cordon, !decreto com percentDecrees/toggleDecrees/onceDecrees e os impostos de decreto PixLevy/levy, !impeachment, !renunciar e decretos que sobram, !golpe). Carregue antes de mexer em presidente, decreto, imposto de decreto ou eleição."
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
  turno vai pra sorte. Quem ganha fica `termLength` no cargo. Os poderes entram pelo `ProfessionOf`, que aplica `presidential()` na
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
  toma posse. Caminho novo que mande alguém pra cadeia precisa passar pelo `arrestShield` (só o
  subornado) e pelo `immunity()`, ou pelo `flagrantOdds` se for flagrante na hora. Tudo some quando o `mandato_ate`
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
  `!empregados`) e o toque de recolher (`curfew()`, tirado da chance de `!roubar`, `!furtar`,
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
  imposto fianca` cai nela pelo mesmo motivo do `iof`, e o `!hospital` e o `!resgate` não pagam) e
  o `cassino`, lido no `PlaceBet`, que só os jogos de cassino usam no lugar do `PlaceStake`
  (bolão, leilão e jokenpo continuam). Ele é um decreto só, com o sinal decidindo: positivo (até
  `casinoDecreeMaxRaise`) aumenta o lucro de quem ganha, e negativo (até `-betTaxDecreeMax`) vira
  uma taxa que vai pro presidente (`casinoLevy`, que inverte o nível): quem perde (ou empata) paga a
  porcentagem em cima da aposta e quem ganha paga em cima de tudo que levou, aposta inclusa
  (`onBet`; ele mesmo não paga; o `!decreto imposto aposta` e o `!decreto taxa aposta` caem no `taxedRun`, que trata o número como taxa e põe o sinal de menos). O
  `PlaceBet` devolve um `wager` com o `Boost` da hora da aposta (`PrizeBoost`: o `decree` do decreto
  e o `luck` da profissão, o `casinoLuck` do Baralho do Zé Pelintra), que fica guardado na aposta (e
  no save da rodada, em `Boost` e `Luck`) e é aplicado no pagamento pelo `Boost.Pay`, com o
  `Boost.Tag` na resposta. O `Boost` também leva o `rescued`, sorteado ali mesmo com o `rescueOdds`
  da profissão (hoje só o Baralho, 20%) e salvo em `Rescued`: na derrota, o `Boost.Rescue` devolve a
  aposta inteira (no `!crash` e na `!corrida`, quem perdeu recebe pelo `payRescues` no fim da
  rodada). O `wager` vem também com o `Levy` da taxa, que o `SettleBet` (ou o `holdBet`, nos jogos
  que seguram a aposta) desconta junto da aposta, na mesma transação. Nos jogos que seguram a
  aposta, a taxa em cima da aposta sai na hora e fica guardada na aposta (`heldLevy`, também no save
  da rodada); se ganhar, o `payPrize` desconta do prêmio o que falta pra completar a porcentagem de
  tudo que levou (`onPrize`). O `tudo`, o `metade` e o `%` já contam a taxa da aposta
  (`stakeAfterLevy`); o dobrar do `!blackjack` não paga taxa. Os de uma vez são `imposto`, `auxilio`
  (`presidentAid` de moeda nova pra cada carteira da conversa), `caixa` (uma `presidentBoxItem`, a
  caixa misteriosa comum, na mochila de cada carteira da conversa), `anistia` (zera `preso_ate` e o
  `crime_em` dentro da `crimeWindow`), `indulto @pessoa` (`presidentPardons` por mandato), `confisco
  @pessoa` (`presidentSeizePercent` do saldo pro presidente) e `acoes <ticker> <pct>` (mexe direto
  no `acoes_precos`, e a reversão do passeio aleatório puxa o preço de volta com o tempo). Decreto
  novo precisa de coluna numa migração, de entrada numa das três tabelas (e no mapa `decrees()`, se
  for de uma vez) e do gancho no comando.
- O `!impeachment` (`impeachment.go`) tira o presidente antes do fim do mandato. O primeiro voto
  abre a votação (`impeachments`, uma por conversa, efêmera e salva com `KeepGame`), presa ao
  presidente da abertura, que fica aberta `impeachmentWindow`. Cada um vota `sim` ou `nao` (sem
  nada é sim) e pode mudar o voto até fechar, com o peso do `ballotWeight` (o voto de deus vale 3).
  Passa com pelo menos `impeachmentQuorum` votos a favor e mais a favor do que contra, e aí o
  `impeach` joga o `mandato_ate` pra agora, então os poderes e os decretos caem sozinhos e o
  `!votar` volta a abrir eleição. A abertura carimba a coluna `impeachment_em` da `presidencias`
  (zerada no `inaugurate`), e o próximo pedido contra o mesmo mandato só sai depois de
  `impeachmentEvery`. Se o presidente da abertura já saiu do cargo quando a votação fecha
  (renúncia ou fim do mandato), o `impeachmentResult` só avisa e não conta nada.
- O `!renunciar` (`renunciar.go`) é o presidente largando o cargo: o `resign` joga o `mandato_ate`
  pra agora e marca `renunciou = 1` na `presidencias`, então os poderes caem na hora e o `!votar`
  volta a abrir eleição, mas os decretos continuam valendo até o próximo `inaugurate`, que zera o
  `renunciou`. Por isso o `walletStore.decrees` e o `presidency` leem a linha com
  `mandato_ate > agora OR renunciou = 1`; com o mandato vencido, o `presidency` devolve `found`
  falso e um `presidency` só com os `decrees` (sem presidente). Os decretos de imposto continuam
  cobrando, mas viram ralo: o `levy` da presidência sai com o `president` vazio, o
  `collected()` do `PixLevy` fica falso, o `payLevy`, o `payPrize` e o `transferWithPix` não
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
  com `mandato_ate` de `coupTerm`, `golpista = 1`, `golpe_em` agora e abusos e decretos zerados
  (o `UPDATE` exige o presidente da abertura ainda no cargo, senão falha com `errCoupVacant`); o
  presidente derrubado só perde o cargo. Se der errado, os golpistas passam pelo `jailCrew` do
  `!assalto`. O `inaugurate` zera `golpe_em` e `golpista`.
