---
name: economia-dinheiro
description: "Mercado e patrimônio da economia (!leilao, !bolao, jogo do bicho !bicho e runAnimalGame, !agiota e dívidas, !emprestimo do banco, !acoes, imposto diário runTaxes, imóveis imovel.go e guardados, !empregados, títulos titulo.go, implantes do !implante em implante.go, obras do !doar). Carregue antes de mexer em algum desses comandos ou nas tabelas deles."
user-invocable: false
---

# Economia: mercado e patrimônio

Leilão, apostas de bolão, empréstimos, ações, imposto e patrimônio.

- O `!leilao` (`leilao.go`) é o leilão do dia: o bot leiloa um item por conversa, e ninguém leiloa
  item da própria mochila. O item sai do `auctionLot`: um rodízio fixo por conversa
  (`auctionRotation`, o `catalog` filtrado pelo `auctionGoods` e embaralhado com a semente do chat,
  andando uma posição por `localDay`), então o item nunca repete de um dia pro outro e só volta
  depois de passar a lista toda. O `auctionGoods` pega épico e lendário que não é ilegal, não é da
  loja e não é caixa de relíquia. Com `auctionRelicChance` (sorteio do `dailyRand`), o dia é de
  relíquia, num rodízio próprio das `relics()`. O lance mínimo é o `auctionOpening`: o `value`, o
  `luxuryPrice` pros `luxuryGoods` (pra não sair mais barato que a `!luxo`) e `auctionRelicPercent`
  (hoje 3000%) do `value` na relíquia, alto de propósito: relíquia não foi feita pra ser comprada, então não baixe esse preço pra ela ficar acessível. O leilão só vira linha na tabela `leiloes` (`chat`, `dia`,
  `item`, `lance`, `licitante`, `fecha_em`) no primeiro lance; antes disso ele é calculado na hora
  (`todaysAuction`). O `bidAuction` desconta o lance, devolve o do coberto e grava tudo numa
  transação só, e lance nos últimos `auctionExtension` empurra o `fecha_em` (que começa na
  meia-noite de BRT), então o leilão de ontem pode seguir aberto uns minutos depois da virada; o
  `loadAuction` pega sempre a linha aberta mais antiga e só cai no de hoje quando não tem nenhuma.
  O `runAuctions`, iniciado junto do `runTaxes`, fecha as linhas vencidas com um
  `DELETE ... RETURNING` (então reiniciar não entrega duas vezes), põe o item na mochila de quem
  ganhou com qualidade normal e anuncia com o `announce` (em todo grupo da economia com o submodo
  `anuncios` ligado, respeitando o `AllowedIn` de quem ganhou). Ninguém recebe as moedas do lance vencedor: é ralo. Dia sem lance não deixa
  rastro.
- O `!bolao` fica no banco (`boloes`, `bolao_opcoes`, `bolao_apostas`): a aposta desconta na
  hora e o `!bolao resultado` divide o pote entre quem acertou, proporcional ao que apostou (a
  sobra do arredondamento vai pra maior aposta); se ninguém acertou ou o bolão é cancelado, todo
  mundo recebe de volta. Fechar, dar resultado e cancelar é só pra quem abriu, admin ou dono.
- O `!bicho` (`bicho.go`) é o jogo do bicho. A aposta desconta na hora e vira uma linha em
  `bicho_apostas` (até `animalBetsPerDraw` por pessoa em cada extração), e o resultado é um só pra
  todas as economias: `bicho_resultados` guarda um milhar (0000 a 9999) por extração (`sorteio`, o
  horário em Unix). As extrações são as `animalDraws` (hora em BRT e nome). O `runAnimalGame`,
  iniciado junto do `runLottery`, grava a cada `core.ReminderTick` o resultado da última extração
  que já passou (`recordAnimalDraw`, que também apaga os resultados mais velhos que
  `animalResultsKept`) e paga as apostas vencidas que o `dueDraws` acha (o mesmo da loteria, com a
  tabela de parâmetro). O `payAnimalDraw` pega o milhar pelo `animalResultOf`, um `INSERT ... ON
  CONFLICT DO UPDATE ... RETURNING`, então cada extração é sorteada uma vez só e reiniciar não muda
  o número; depois apaga as apostas com `DELETE ... RETURNING` e credita os prêmios na mesma
  transação, e o aviso sai pelo `announce` com os ganhadores marcados. Os tipos de aposta ficam em
  `animalBetKinds` (grupo, dezena, centena e milhar), cada um com o pagamento (`odds`) e a aposta
  máxima (`limit`), e só o `id` vai pra coluna `tipo`: nunca renomeie um. O grupo sai da dezena do
  milhar (`animalOf`, 4 dezenas por bicho, e a 00 é da vaca) e é a posição em `animals`, então
  nunca reordene a lista. O que não acerta fica com a banca (ralo). O bicho é clandestino: não passa
  pelo `PlaceBet`, então o decreto `cassino` não pega nele, nem os bônus de
  cassino de profissão e de item. A única vantagem é o `animalLuck` da profissão (hoje o bicheiro),
  que soma porcentagem no lucro como a `luck` do `PrizeBoost` e fica gravado na aposta (`sorte`),
  então trocar de profissão depois não muda aposta já feita.
- O `!agiota` (`agiota.go`, com `!emprestar` de alias) é o empréstimo entre pessoas, e o agiota é
  quem empresta; o empréstimo do banco é outro comando, o `!emprestimo` (abaixo). Quem pede
  manda `!agiota @pessoa <valor> [dias]` e o pedido fica em `debtRequests` (um por quem empresta, na
  mesma chave do `itemOffers`, então o `!aceitar` e o `!recusar` atendem os dois) até a pessoa
  responder. Aceito, a dívida vai pra tabela `dividas` (uma por par devedor/credor) e os juros
  compostos são calculados na leitura (`debt.accrued`): `juros` por dia até o `vence_em` e
  `juros_atraso` depois. Com a dívida vencida, o credor pode `!matar`, `!prender`, `!sequestrar` ou
  `!processar` quem deve (`rivalsWithDebt`) mesmo sem a profissão certa e sem `crime_em`; é cobrança, então
  não marca crime, não tem chance de ir preso na hora e o `!prender` não zera o `crime_em` do
  devedor. A imunidade do presidente continua valendo no `!prender`.
  - A profissão `agiota` mexe nesse sistema pelos campos `lend*`, `collect*` e `garnish` da
    profissão de quem empresta. Os juros (`loanRates`, `lendRate`/`lendLateRate` somados no
    `debtRate`/`debtLateRate`) são lidos no pedido, guardados no `debtRequest` (e no save) e
    gravados na linha da `dividas`, então trocar de profissão depois não muda dívida antiga. O
    `collectBonus` soma na chance do `!matar` e do `!sequestrar` de cobrança (`collectionEdge`),
    o `collectEvery` troca o `debtArrestEvery` do `!prender` de cobrança e o `collectVotes` soma
    votos pra culpado no `!processar` por calote. O `garnish` desconta uma porcentagem do que o
    devedor atrasado ganha no `!trabalhar` e paga a dívida na mesma transação (`salaryLien`, lido
    em paralelo no `trabalhar`, e `garnishPay` → `settleDebt`, o mesmo miolo do `repayDebt`); com
    mais de um credor desses, só o de vencimento mais antigo desconta.
- O `!emprestimo` (`emprestimo.go`) é o empréstimo do banco: o bot empresta moedas novas e as
  moedas pagas de volta somem (ralo). Fica na tabela `emprestimos` (refeita na migração `00080`),
  um por pessoa por conversa, e reaproveita o `debt` do `!agiota` sem credor (o `lender` fica
  vazio), então os juros compostos também saem do `debt.accrued`: `loanRate` por dia até o
  `vence_em` e `loanLateRate` depois. O limite (`loanLimit`) é o `max` do `!trabalhar` da
  profissão vezes `loanLimitShifts`, entre `loanLimitFloor` e `loanLimitCap`, então o presidente e
  os imóveis com `payBonus` pegam mais. Com o prazo vencido, o `runLoanSeizures` (iniciado junto do
  `runAuctions`) penhora o saldo a cada `loanSeizeEvery`: o `seizeLoan` empurra o `penhora_em`
  antes de cobrar (então reiniciar não penhora duas vezes), tira até a dívida toda pelo
  `settleLoan` (o mesmo miolo do `!emprestimo pagar`) e anuncia com o `announce`, em todo grupo da
  economia com o submodo `anuncios` ligado e respeitando o `AllowedIn` de quem deve. Saldo zerado não gera aviso. O `!emprestimo` não tem
  nada de cobrança na marra: quem empresta é o banco, então não libera `!matar`, `!prender` nem
  nada parecido.
- As `!acoes` guardam o preço por conversa em `acoes_precos` (os saldos também são por conversa);
  o passeio aleatório avança um passo a cada `stockStepEvery` quando alguém consulta, e as
  posições ficam em `acoes_posicoes`. As empresas vivem em código (`stockCompanies` em
  `acoes.go`) e só o `ticker` vai pro banco: nunca renomeie um ticker existente.
- O imposto (`imposto.go`) roda no `runTaxes`, iniciado junto do `runBirthdays`: a partir de
  `taxHour` (BRT), cada economia com o modo `economia` ligado em pelo menos um grupo é marcada na tabela `impostos`
  (`chat`, `dia`) antes de cobrar, então reiniciar não cobra duas vezes. Cobra as faixas marginais
  de `taxBrackets` (`wealthTax`) mais o IPTU do imóvel, menos o `taxCut` do melhor contador
  contratado no `!empregados` (lido no `taxPayers` junto da carteira), limitado ao saldo, e anuncia com o
  `announce` (respeitando o `AllowedIn` de quem mais pagou). O `!transferir` e a venda do `!vender @pessoa`
  não têm taxa fixa; a única taxa de transferência é a do decreto `pix`.
- Os imóveis vivem em código (`estates` em `imovel.go`) e só o nível vai pra coluna `imovel` da
  `carteiras`. Sobe um nível por vez, pagando o preço cheio, e cada nível dá `guard`,
  `dailyBonus`, `payBonus` (porcentagem a mais no `min`/`max` do `!trabalhar`, aplicada no
  `housed`) e cobra IPTU (`tax`) no imposto diário. Nunca reordene a lista: o nível é o índice.
  O `staff` de cada nível é o número de vagas de empregado (`!empregados`); sem imóvel, nenhuma.
  O `storage` de cada nível é quantos itens cabem guardados em casa (`guardados.go`): o
  `!imovel guardar`, o `!imovel tirar` e o `!imovel guardados` mexem na tabela `guardados`, que tem
  as mesmas colunas do `inventarios` (qualidade na chave e `usos`). As funções de mochila têm uma
  versão com a tabela de parâmetro (`loadStacks`, `takeGradesFrom`, `addGradesTo`), e o `shiftItems`
  passa item entre as duas levando a qualidade e o desgaste (`usos`) junto, pra guardar e tirar não
  zerar o desgaste da ferramenta. Item guardado não tá na mochila, então nada que lê o `inventarios`
  enxerga ele: não é furtado, não é apreendido, não entra nas vendas em lote e não dá efeito nenhum
  (passivas, arma, ferramenta) até voltar pra mochila. O nível do imóvel só sobe, então o espaço
  nunca encolhe embaixo de quem já guardou.
  A maldição Casa assombrada (skill `encantos-e-zoeiras`) mexe no imóvel: inverte o `guard`, o
  `dailyBonus` e o `payBonus` dele, o IPTU vem em dobro no imposto e o `!imovel` mostra quem
  mandou o espírito e quanto custa o `!imovel exorcizar`. A relíquia Carranca do Velho Chico
  segura o espírito enquanto tá na mochila e ainda faz o imóvel render `carrancaBoost` por cento a
  mais (mesma skill).
  O `vault` de cada nível é quanto cabe de moeda no cofre de casa (`cofre.go`, coluna `cofre_casa`
  da `carteiras`, que não é o `cofre` das `igrejas`): o `!imovel depositar` passa do `saldo` pro
  cofre (cortando no espaço que sobra), o `!imovel sacar` faz o contrário e o `!imovel cofre`
  mostra. As duas pontas passam pelo `shiftVault`, um `UPDATE` só condicionado ao saldo e ao
  tamanho do cofre. É o mesmo princípio dos guardados: moeda no cofre não tá no `saldo`, então nada
  que lê o `saldo` enxerga ela (`!roubar`, `!quadrilha`, multa da cadeia, enterro, `!castigo`,
  imposto, penhora do `!emprestimo`, `!ricos`). Por isso os tamanhos são pequenos de propósito (uma
  fração pequena do preço do imóvel): o cofre protege uma reserva, não a fortuna toda.
- Os empregados (`empregados.go`) são fictícios e vivem em código (`staff`), e só o `id` vai pra
  coluna `tipo` da tabela `empregados`, uma linha por empregado contratado (cada um com o próprio
  `coletado_em`, então contratar outro do mesmo tipo não herda as horas dos antigos). Cada tipo
  faz uma extração (`activity`) sozinho a cada `every`, ajustado pelo decreto `extracao`
  (`pace`), e guarda até `stock` achados. Nada roda em segundo plano: o `!empregados coletar`
  calcula as rodadas desde o `coletado_em` (`ready`), cobra o `wage` de cada rodada do saldo e
  sorteia o `roll()` da atividade nerfada pelo `shift`; as rodadas que o saldo não paga viram
  greve e se perdem. O `shift` tira do `loot` tudo que é lendário ou acima, `unboxable()` e
  Bússola do relicário (`relicLure`), e o empregado não usa ferramenta, equipamento nem
  profissão (sem quebra, sem machucado). O aprendiz puxa a qualidade pra baixo (`gradeLuck`
  -1) e o veterano escolhe o melhor de dois achados (`luck` 1). O `price` é calibrado pra se
  pagar em **72 horas** produzindo sem parar (o líquido por rodada, com o `wage` em 30% do que
  a rodada rende em média na venda), então mexer no `loot` de uma atividade pede recalibrar os
  empregados dela. O caçador de bússola é a exceção: não tem `activity`, e o `quarry` faz o
  `shift` montar um `loot` só com a Bússola do relicário (peso 1) e a mão vazia (peso `misses`,
  hoje 1 em 250), bem pior que o caçador de bússola jogador (1 em ~95 a ~133 a cada 5 minutos,
  nas quatro extrações). O preço dele não segue as 72 horas pelo valor de venda, que daria
  quase nada: o custo de verdade é a vaga do imóvel que ele ocupa.
  O apostador e o corretor não têm `activity`: o `payouts` sorteia (`bet`) quantas vezes o
  `wage` voltam em moedas a cada rodada, e o `harvest` soma em `coins`, creditado no saldo depois
  de descontar o salário (o ganho da coleta não paga o salário dela). O decreto `extracao` não
  muda o ritmo deles. O apostador tem muita variação (perde tudo na maioria das rodadas, com
  prêmio de até 50x) e o corretor pouca (nunca volta com menos do que cobrou); o `price` dos dois
  segue as mesmas 72 horas, pelo líquido médio da tabela (`odds`). O contador (`taxCut`) e o
  enfermeiro (`care`) são passivos (`passive()`): não têm `every` nem `wage`, ficam fora do
  `!empregados coletar` e valem só de estar contratados, sempre o melhor de cada tipo
  (`strongest`). O contador corta a porcentagem do imposto e do IPTU no `dueTax`, e o enfermeiro
  acelera a saúde no `regenEvery`. Consulta que precisa dos tipos contratados junto da carteira
  usa o `staffKinds` (um `GROUP_CONCAT`, com a carteira no alias `c`) e o `kindsOf`. Nunca renomeie um `id`: o empregado com o id antigo some. O `!contratar` é um
  atalho do `!empregados contratar`: os dois chamam o `hire`, que só muda quantas palavras pula
  antes do nome do empregado.
- Os títulos (`titulos` em `titulo.go`) são diferentes do `!apelido`: cada um tem um dono só por
  conversa (tabela `titulos`, PK `chat`, `titulo`), e o equipado fica na coluna `titulo` da
  `carteiras`, que aparece no `!saldo` e no `!ricos` (`titleBadge`). Tomar de alguém custa
  `titleStealPercent` do que a pessoa pagou e devolve `titleRefundPercent` pra ela (`dethrone`).
  O `claimTitle` confere se o dono ainda é o mesmo e falha com `errTitleChanged` se mudou. Nunca
  renomeie um `id`. Cada título dá uma vantagem leve no campo `bonus` (um `flex`), e só o
  equipado vale: o `standing` lê a coluna `titulo` e o `ProfessionOf` aplica o `titled` antes do
  `charmed`. A vantagem é de um de três jeitos: um efeito pequeno direto na profissão, uma
  maldição pra lançar (`titleCurse`, que soma na `faith` e pega com pelo menos `titleHexChance`)
  ou um tipo de abuso de poder (`titleAbuse`, que soma no `abuses`). O `castersOf` também lista
  os títulos. Título não dá bênção: como dá pra lançar em si mesmo, a bênção vira um efeito
  permanente bem mais forte que o resto. Mantenha as vantagens fracas e crescendo com o preço
  (compare com os itens de luxo que dão o mesmo efeito), porque o título passa de mão em mão.
- Os implantes (`implants` em `implante.go`) vivem em código, e só o `id` vai pra tabela
  `implantes` (`chat`, `pessoa`, `parte`, `implante`, `instalado_em`, `pago_ate`, `desgaste`, PK
  `chat`, `pessoa`, `parte`), então cada parte do corpo aguenta um implante só. As partes ficam em
  `bodyParts` (cérebro, olhos e braço), indexadas como as `areas`, e o `key` delas vai pra coluna
  `parte`. Nunca renomeie um `id` nem um `key`. Fora do corpo, o implante é item: o
  `implantGoods` monta um item lendário por implante (mesmo `id`, `value` = `price`, campo `graft`
  apontando pro implante) e ele entra no fim do `catalog`. Esse item é `unboxable()`, não tem
  qualidade, não vai pro `!leilao` e tem o tipo `Implantes`; só chega pelo `!mercadonegro`, um por
  dia (skill `economia-itens`). O `!implante instalar` e o `!usar` (`implantSurgery` → `operate`)
  recusam parte ocupada (`errPartTaken`), tiram o item da mochila (`errNoItems` se não tiver) e
  sorteiam a rejeição pelo `risk()`, um `hazard` com chance `reject` e dano `implantRejectHarm`
  aplicado pelo `hurt` da saúde, que pode matar. Rejeitado, o item se perde; instalado, ele vem
  pago por `implantUpkeepGrace`. O desgaste é o `usos` do `inventarios` (o mesmo `MAX` das
  ferramentas, que o `moveItems` leva junto no `!dar`, no `!vender @pessoa` e no `!furtar`) na
  mochila e a coluna `desgaste` no corpo, e passa de um pro outro na instalação.
  Toda saída do corpo (`!implante tirar` e o arrancado do `!matar`) passa pelo `shelveImplant`,
  que soma 1 no desgaste e devolve pra mochila, ou quebra o implante quando chega em
  `implantWearMax`. O efeito é um `flex` (`bonus`): o `ProfessionOf` lê o corpo em paralelo
  (`walletStore.body`) e aplica o `implanted` logo depois do `adorned`, só nos implantes com
  `pago_ate` no futuro. Os outros vão pro `unpowered` e aparecem desligados nos `perks()`. Nada
  roda em segundo plano: o `!implante manutencao` (`maintain`) cobra o `upkeep` diário
  proporcional ao que falta pra deixar todos pagos por `implantUpkeepMax` (ralo). Implante com
  `illegal` é clandestino: na mochila é contrabando, e o
  `walletStore.arrest` (`!prender` e `!denunciar`) arranca ele pelo `seizeImplants`, junto do
  `seizeContraband`; a batida do `!mercadonegro` não. Quando o `!matar` dá certo, o `ripImplant`
  tem `implantRipChance` (baixa de propósito) de arrancar um implante qualquer da vítima e pôr na
  mochila de quem matou pelo `shelveImplant`. Pra calibrar o preço, compare com o item de luxo de
  efeito parecido (que custa `value` vezes `luxuryMarkup` e não tem risco nem manutenção): o
  implante pode ser mais barato ou mais forte porque pode ser rejeitado, desligar, desgastar e ser
  arrancado. O preço de compra é o do mercado negro do dia, sobre o `price`.
  O Coração do Corpo-Seco (relíquia passiva, `corpoSecoItem`) protege o corpo de quem tem ele na
  mochila, checado dentro da transação pelo `holdsItem`: o `operate` ignora a rejeição (e avisa
  no texto com o `spared`), o `ripImplant` não arranca nada da vítima e o `seizeImplants` não
  tira os clandestinos instalados. Os clandestinos que tão na mochila continuam sendo
  contrabando pro `seizeContraband`, e a manutenção não muda. O `sealedBody` da profissão só
  serve pro texto no `implantPerks`.
- As obras do `!doar` (`doar.go`) seguem `publicWorks` em ordem por conversa (`obras` guarda o
  nível e o arrecadado, `obras_doacoes` quanto cada um deu em cada obra). Depois da última vêm
  monumentos com a meta dobrando (`workAt`). Doação que passa da meta segue pra próxima, e a obra
  pronta homenageia quem mais doou pra ela (`obras_concluidas`). `!doar @pessoa` continua sendo o
  `!dar`.
