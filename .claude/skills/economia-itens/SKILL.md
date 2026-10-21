---
name: economia-itens
description: "Itens da economia (catalog em itens.go, inventarios, raridades, relíquias, míticos e primordiais, Meteorito do Bendegó em bendego.go com Rocha celeste, fragmentos, cratera e campo gravitacional, !primordial, qualidade.go, ferramentas de extração e a calibração do tempo de se pagar, equip/loot/roll, tipos itemCategories, números #12, vendas em lote do !vender, feira entre jogadores do !feira com a tabela feira_anuncios, favoritos do !favoritar, !loja e ofertas do dia, !luxo, !mercadonegro, !petshop, resets/limiter do !usar, horta do !plantar/!colher/!horta com sementes, canteiros e a tabela plantacoes). Carregue antes de mexer em item, loja, mochila, extração (!pescar, !minerar, !vasculhar, !catar), horta ou venda."
user-invocable: false
---

# Economia: itens

Catálogo, raridades, qualidade, lojas e vendas de item.

- Os itens da economia ficam na tabela `inventarios`, mas o catálogo vive em código (`catalog`
  em `itens.go`) e só o `id` de cada item vai pro banco. Nunca renomeie um `id` existente: os
  itens com o id antigo somem das mochilas. As propostas do `!vender @pessoa` são efêmeras
  (`itemOffers`) e só trocam item e moedas na hora do `!aceitar`.
- Relíquia é uma raridade própria (`relic`, acima da `legendary`), não uma lendária: ferramenta
  relíquia nunca quebra (`sturdy: 100`), tem `moreRare: 100` e nunca volta de mão vazia (`alwaysFinds`, que tira a entrada vazia do loot no `equip`); ferramenta lendária tem
  `sturdy: 97` e `grace: 100`. Item com `rarity: relic` (hoje a
  Vara de Iemanjá, a Picareta de Serra Pelada, a Rede do Curupira, o Saco do Homem do Saco, a
  Vela do Negrinho do Pastoreio, a Excalibur e as passivas Muiraquitã, Gorro do Saci, Chapéu de Lampião, Escamas do Boitatá, Coroa
  de Dom Pedro, Santo Graal, Baralho do Zé Pelintra, Espelho da Iara, Lamparina do Lobisomem, Foice da
  Comadre Morte, Livro de São Cipriano, Chapéu do Boto, Coração do Corpo-Seco, Quepe da Volante, Carranca do Velho Chico e Búzios encantados) fica fora da loja, das atividades, da `!luxo` e das caixas e baús comuns. Ele só sai
  de caixa com `relicOdds` (hoje o Relicário perdido), que no `roll` troca o prêmio por uma
  relíquia aleatória nessa chance, do `!leilao` nos dias de relíquia e do `!criar` e do `!milagre` de deus; depois passa de mão em
  mão, mas nunca pelo `!furtar` (o `theftChoice` tira as relíquias do sorteio e o furto mirado
  numa relíquia é recusado). A caixa com `relicOdds` também não sai de outra caixa e só cai nas extrações (`loot` das
  `activities`, peso 2). Equipamento com `relicLure` (hoje a Bússola do relicário, que sai de
  caixa comum e, com peso 1, das extrações) multiplica esse peso no `equip` (`lureRelics`) e, com `spent`, é gasto na próxima
  extração. Sem equipamento com `relicLure`, o `equip` passa toda a `luck` (ferramenta, acessórios
  e `craft` da profissão) pro `plainLuck`, então a sorte só ajuda a achar a caixa de relíquia com a
  bússola equipada; sem ela, só o sorteio principal acha. O filtro de metal do Ímã tira a caixa de relíquia do sorteio, mesmo com a bússola. O
  `randomItemOf` pula tudo que é `unboxable()` (luxo, ilegal, relíquia, mítico e
  caixa de relíquia). Ferramentas (`gear` com `tool`) não somam entre si: se a pessoa tem mais de
  uma da mesma atividade, o `equip` só aplica a de maior `value` (`bestTool`), e as outras ficam
  paradas na mochila. Os acessórios (`gear` sem `tool`, como isca, capacete e bússola) continuam
  valendo junto com ela. A sorte de acessório com `skipsRelics` (a isca) vai pro `plainLuck` do
  `roll` mesmo com a bússola, e nesses sorteios extras a caixa de relíquia é ignorada. A Excalibur é arma legal: o `bestWeapon` lê toda arma do `catalog` (não só as
  ilegais), mas ela não é apreendida porque o `seizeContraband` só pega item `illegal`. O
  `!vasculhar` não tem ferramenta base, então a relíquia dele (o Saco do Homem do Saco) é acessório,
  sem `tool` e sem `sturdy`. A Vela do Negrinho do Pastoreio também é acessório, mas vale nas
  quatro extrações: não substitui a ferramenta, só soma sorte e tira a mão vazia do sorteio
  (`alwaysFinds`). As relíquias passivas usam o mesmo `flex` dos itens de luxo (o
  `passives` lê qualquer item com `flex`), mas nunca levam `keep`: o `luxuryGoods` põe na `!luxo`
  todo épico ou acima com `keep` que não é da loja. A Lamparina do Lobisomem é a única que depende
  da hora: o `apply` dela só mexe na profissão quando o `werewolfHour` (0h às 6h em
  `core.BrazilTime`) dá verdadeiro, e aí soma em quase tudo (crimes, com o `killChance` liberando o
  `!matar` pra qualquer um, proteção, fuga, ganho e espera do `!trabalhar`, `haste`, achados,
  `!daily`, poupança, cassino, caixas, refém e `!acoes`). Como o `ProfessionOf` monta a profissão
  a cada comando, o efeito liga e desliga sozinho, sem estado guardado. As relíquias que mexem
  num sistema de outra skill ficam documentadas lá: o Chapéu do Boto na `economia-fama`, o
  Coração do Corpo-Seco nos implantes da `economia-dinheiro`, o Quepe da Volante no procurado do
  dia da `economia-crime`, a Carranca do Velho Chico na Casa assombrada da `encantos-e-zoeiras` e os
  Búzios encantados na `economia-eventos`.
  Cada efeito novo precisa de uma linha própria nos `perks()` (ou no `famePerks` e parecidos),
  porque o `luxuryPerks` só lista o nome da relíquia com "já conta nas vantagens acima".
- Mítico é outra raridade própria (`mythic`, entre a `legendary` e a `relic`): são itens de
  extração que só servem pra vender (sem `use`, sem `keep`, sem efeito), dois por atividade, com
  peso 1 no `loot`. Eles só saem com relíquia ou acima: o `equip` tira os míticos do `loot` quando
  nenhum item equipado naquela extração é relíquia ou primordial e a atividade não tem `mythics`
  (a cratera do Bendegó), então a ferramenta relíquia da atividade (ou o Saco do Homem do Saco), a
  Vela do Negrinho do Pastoreio e o Meteorito do Bendegó liberam eles, e o `source` da atividade
  mostra quais itens liberam no "Onde conseguir". São `unboxable()`, então não saem de caixa nem de
  baú comum (só da Rocha celeste, a única caixa com peso de mítico), mas têm qualidade e podem ser
  furtados. Na venda pro mercado, o `marketPercentOf`
  paga pelo menos `mythicSalePercent` (hoje 90%) por eles, ou o `salePercent` da profissão se for
  maior; ele é aplicado dentro do `marketPayoutOf` e do `marketPayout`, então vale no `!vender`
  (todas as formas), no valor de mercado da `!mochila` e no `!item`.
  O `!vender inuteis` vende mítico junto com o resto.
- Primordial (`primordial`, ⚫) é a raridade mais alta, acima da `relic`, com tipo próprio
  (⚫ Primordiais, o primeiro do `itemCategories`). Ela herda as travas da relíquia pelas
  comparações com `>=`: é `unboxable()`, não tem qualidade (`gradable()`), não se furta (`!furtar`
  mirado e `theftChoice`), libera os míticos quando tá equipada na extração e fica fora do
  `findable` dos eventos. O que olha só a relíquia com `==` continua sem pegar ela: o `relics()`
  (Relicário perdido e dias de relíquia do `!leilao`) e o tipo 🏛️ Relíquias. O "Onde conseguir"
  dela sai do `primordialSources` (no `sources`), e o `effectText` tem aviso próprio pra prêmio
  primordial. Item primordial nunca leva `keep`, senão o `luxuryGoods` põe ele na `!luxo`. O
  `loot` das caixas é lido pelo `lootWeight`, porque a maioria só tem posição até a lendária.
  Cada primordial tem um poder que vale só de ter (`passive`) e uma habilidade (`active`), os dois
  no `primalPowers` (`bendego.go`), que também liga o subcomando do `!primordial <item>` ao `run`
  dela; o `effectPerk` mostra o `gear` mais os dois textos (`primalPower.perk`). Primordial novo
  com habilidade entra nesse mapa. Hoje são três, todos em `bendego.go`:
  - A Rocha celeste (`skyRockID`) cai em qualquer extração com uma relíquia de extração equipada
    (`extractionRelic`: relíquia com `gear` de uma atividade só, então a Vela do Negrinho do
    Pastoreio não conta). Ela não tá no `loot` das atividades: é o `starfall` do `equip` que liga
    uma chance fixa de `starfallOdds` (0,1%) no `pick`, antes do sorteio normal, então nem a sorte
    da ferramenta nem o `boostRare` mudam essa chance, e as ferramentas relíquia não precisam de
    recalibragem. Ela é caixa: `loot` só de lendário e mítico (o `randomItemOf` aceita qualquer
    mítico, que de outro jeito nunca sai de caixa) e, com `shardOdds` (50%), troca o prêmio pelo
    `shard` dela, o Fragmento do Bendegó.
  - O fragmento tem `forms` (o item que ele monta) e `shards` (quantos precisa). A montagem é no
    `addGradesTo`, só na mochila: toda entrada de item passa pelo `assembleShards`, que troca cada
    `shards` fragmentos por um Meteorito do Bendegó na mesma transação, venha de onde vier (caixa,
    `!dar`, `!vender @pessoa`, `!criar`, `!milagre`, tirar do imóvel). Ele grava o aviso em
    `montagens_avisos`, e o `announceAssemblies` (`AddAfterMessage`) manda no grupo depois da
    resposta do comando. Fragmento guardado no imóvel não monta.
  - O Meteorito do Bendegó (`bendegoID`) é acessório das quatro extrações com `pull` (o `equip`
    soma em `kit.pull`, e o `gather` sorteia esses itens a mais com `find`, que não quebra
    ferramenta) e `alwaysFinds`, então toda extração traz 2 itens; a gravidade aparece no
    `report`, e os avisos de mítico, Relicário e Rocha celeste saem do `findNotes`, uma vez cada.
    O Campo gravitacional: no `!trabalhar` (`walletStore.work`), o `pullGravity` tira
    `gravityPercent` (3%) do ganho de quem não tem Bendegó, se alguém da economia tem, e soma em
    `campo_gravitacional`; o `runGravity` reparte o acumulado entre os donos de hoje a cada
    `gravityEvery` (6h, contadas da primeira puxada depois do último pagamento) e avisa pelo
    `announce`. A habilidade é a cratera (`!primordial bendego`, espera `cratera_em` de 24h): fica
    `craterOpen` (1h) no `sync.Map` `craters` do grupo, salva com `KeepGame` (`craterKind`). No
    `!minerar` desse grupo, o `crater.dig` liga `mythics` (libera mítico sem relíquia) e soma
    `craterLuck` no `plainLuck`, e o `payToll` cobra `craterToll` (20%) do valor de mercado de cada
    achado, até onde o saldo der, e passa pro dono (o próprio dono não paga). A cratera não dá a
    chance da Rocha celeste a quem não tem relíquia.
- Uma extração pode fazer várias tentativas de uma vez: quem ficou mais tempo parado ganha uma
  tentativa por espera que passou, até `backlogSpan` (turnos acumulados, skill `economia-base`). Cada
  tentativa gasta a ferramenta e os consumíveis como se fosse um comando separado (quebra, `grace`,
  isca, bússola...), então a calibração abaixo continua valendo por tentativa.
- O `!extracao` (`extracao.go`) roda as quatro `activities` em sequência, cada uma pelo mesmo
  `b.extract` do handler `gather` (espera, ferramenta, missão, carreira e ferimento de cada uma), e
  junta os relatórios numa resposta só, com o aviso do decreto uma vez no fim. É em sequência porque
  as quatro mexem na mesma mochila (Vela, bússola, iscas). O atalho é mais fraco que os comandos
  separados de propósito: o `bundled` multiplica a espera de cada atividade por `bundlePacePercent`
  (150%) antes do `extract`, e como o `backlog` divide o tempo parado por essa espera, quem usa o
  `!extracao` também junta menos tentativas por hora. A espera maior só vale pro atalho: o carimbo
  é o mesmo, então o `!pescar` sozinho continua liberando na espera normal. Se a pessoa morre num
  ferimento, as extrações seguintes não rodam e nenhum evento sai; senão, o `stirEvent` roda uma
  vez só, com uma das extrações que rodaram sorteada. Extração nova entra sozinha, porque ele
  percorre o `activities`.
- As ferramentas de extração que não são relíquia (rara, épica e lendária de `!pescar`,
  `!minerar` e `!catar`) têm um tempo padrão de se pagar, hoje uns **130 minutos** (de 125 na
  rara a 155 na lendária, numa simulação com todos os sorteios), contando o que a
  pessoa recebe vendendo no `!vender`, não o valor cheio dos itens. O tempo é o `value` da
  ferramenta dividido pelo líquido por hora: o valor médio do melhor item entre `luck + 1`
  sorteios do `loot` montado pelo `equip` (item `gradable()` vezes a qualidade média da melhor de
  `grade + 1` qualidades), vezes o `marketPercentOf` do item (o `marketSalePercent`, ou o
  `mythicSalePercent` nos míticos, sem o `salePercent` de profissão), menos `breaks` vezes o `value` cheio da ferramenta (a reposição sai pelo preço da
  loja), vezes os usos por hora da espera. O ajuste é nos atributos do `gear` (`luck`,
  `fewerJunk`, `moreRare`), não no preço nem na espera: o `faster` é o mesmo pra toda ferramenta
  de extração da mesma raridade, hoje 0 na rara (12 minutos, igual à base), 40 na épica (7,2), 50
  na lendária (6) e 80 na relíquia (2,4). O `grade` também é fixo por raridade: 1 na rara, 2 na
  épica, 3 na lendária e 4 na relíquia. Ferramenta nova ou mudança no `loot` de uma
  atividade precisa recalibrar as ferramentas dela pra esse tempo. As ferramentas base (Vara de
  pesca, Picareta e Rede de insetos) ficam de fora e se pagam em uns 60 minutos. Como o `loot`
  da mina vale mais, as picaretas têm um `luck` a menos que a vara e a rede da mesma raridade.
  As relíquias de extração que fazem o papel da ferramenta (Vara de Iemanjá, Picareta de Serra
  Pelada, Rede do Curupira e o Saco do Homem do Saco, que é acessório porque o `!vasculhar` não
  tem ferramenta base) seguem a mesma conta com outro tempo, hoje **36 minutos**, usando o `value`
  delas como preço; como nunca quebram, não têm custo de quebra. Os míticos entram nessa conta
  (só elas acham), então mexer em mítico também pede recalibrar as relíquias de extração. A Vela
  do Negrinho do Pastoreio fica de fora, porque só soma em cima da ferramenta que a pessoa já tem.
- O `value` dos achados de extração que só servem pra vender (sem efeito no `effectPerk`, sem
  `gift`, sem `loot`, sem `gear`, sem `seed` e abaixo de mítico) segue uma escala que favorece quem
  tá começando: comparado com o resto do `catalog`, o comum vale uns 50% a mais, o incomum 40%, o
  raro 30%, o épico 20% e o lendário 10%. Achado novo entra na mesma escala, senão a
  calibração das ferramentas acima sai do lugar.
- Os acessórios de extração da `!loja` cobrem os buracos de cada atividade: sorte fixa (Mapa
  geológico, Luva de borracha, Molinete, Guia de campo de insetos), sorte gasta a cada tentativa
  (Isca de minhoca, Garrafa de melado, Pólvora de garimpo), espera menor (Capacete com lanterna,
  Armadilha luminosa, Barco a remo, Carrinho de mão), menos lixo (Peneira de entulho e a Mochila de
  trilha, que vale nas quatro), mais raro (Detector de metais, oferta do dia, no `!minerar` e no
  `!vasculhar`) e qualidade (Lupa de joalheiro). Os novos ficam no `gatherGoods` (`apetrechos.go`),
  o último bloco do `catalog`. Acessório não entra na calibração das ferramentas: ele soma por
  cima e se paga em algumas horas de extração com a ferramenta base, e mais rápido com as melhores.
- O tipo de cada item (`itemCategories` em `itens.go`) não é um campo: é a primeira regra da lista
  que bate com os campos do item (`pet`, `arms`, `heist`, `gift`, `networks`, o loot das `activities`...), e o
  que não bate em nada vira `Raridades`. O `Contrabando` pega o ilegal que não é arma nem droga, e
  o `Vantagens` pega os itens com `flex` que não são relíquia, luxo nem ilegais. O `Aparelhos` pega
  os itens com `networks` (Celular e Computador), que liberam as redes do `!postar` (skill
  `economia-fama`) e não têm qualidade (`gradable()`). O `!item`, o `!item <tipo>`, a `!mochila` e a `!loja`
  agrupam por ele (`categoryGroups`), com a raridade na bolinha da linha. Item novo cai no tipo
  sozinho; tipo novo é só entrar na lista, na posição certa, porque a ordem decide o empate.
  A linha da `!loja` e do `!petshop` (`shopLine` em `loja.go`) mostra a bolinha, o nome da
  raridade, o preço e um resumo curto (`brief`), nunca o `perk()` inteiro, que fica só no
  `!item <nome>`. O `brief` sai dos campos do item, na mesma ordem do `effectPerk` (o `gearBrief`
  resume o `gear` em poucas palavras), e item sem efeito (comida, presente) fica sem resumo. Os
  `flex` usam o `flex.brief` quando o `effect` é comprido demais pra uma linha, senão o próprio
  `effect`; item de loja novo com `flex` de efeito longo precisa de `brief`.
  O `!vender tipo <tipo>` (ou `categoria`) vende tudo o que a pessoa tem daquele tipo, menos os
  ilegais, pelo mesmo `sellBatch` do `!vender #12`. A palavra `tipo` é obrigatória porque vários
  nomes de tipo já acham item sozinhos (`!vender pesca` vende a Vara de pesca).
  As vendas em lote de uma palavra só ficam no mapa do `parseBulkSale`, cada uma com o filtro
  (`pick`) e a dica, e todas passam pelo mesmo `sellBatch` do `!vender #12`:
  - `!vender inuteis` (ou `inutil`, `sobras`): tudo o que não tem `perk()` (o "Pra que serve" do
    `!item`) e não tem `gift`, de qualquer raridade.
  - `!vender bugigangas` (ou `bugiganga`, `tralhas`): tudo o que não tem `effectPerk()`, que é o
    `perk()` sem a parte do `usePerk()` (o `!usar` que só manda texto, com ou sem `keep`). Então
    entram também comida, presente e colecionável. Os bichinhos (`pet`) ficam de fora.
  - `!vender duplicados` (ou `duplicadas`, `repetidos`, `repetidas`): `quantity - 1` de cada item.
    Como o `takeGrades` tira da pior qualidade pra melhor, quem fica na mochila é o de melhor
    qualidade.
  Os ilegais nunca entram. Efeito novo precisa ir pro `effectPerk()`, e não pro `usePerk()`,
  senão o item vira bugiganga (ou inútil) e é vendido.
- Os favoritos (`favoritar.go`, tabela `favoritos` com `chat`, `pessoa` e `item`) marcam um item
  inteiro, não uma qualidade, e a marca fica mesmo se o item sair da mochila. O `!favoritar <item>`,
  `#12 #40` ou `tudo` (só o que tá na mochila) e o `!desfavoritar` (o mesmo, sobre os favoritos)
  passam pelo `pickFavorites`. Favorito não sai em venda nenhuma: o `heldForSale` (que troca o
  `heldItem` no `!vender <item>`, no `!vender @pessoa` e no `!mercadonegro vender`) recusa com o
  `favoriteRefusal`, e as vendas em lote recebem os favoritos no `salePick` (o `sellBatch` lê
  junto com a mochila pelo `shelf`): o `#12` dá problema, e o `tipo`, os `inuteis` e as
  `bugigangas` deixam eles de fora com o aviso do `favoritesKept`. A exceção é o
  `!vender duplicados`, que ignora os favoritos e vende as cópias, sempre deixando 1. O `!dar`, o
  `!usar` e o `!furtar` não olham os favoritos. A `!mochila` mostra ⭐ do lado (`favoriteMark`).
  Venda nova de item precisa passar pelo `heldForSale` ou receber os favoritos.
- A feira (`feira.go`, tabela `feira_anuncios`) é o mercado entre jogadores: o `!feira vender
  <item> [quantidade] <preço>` anuncia pra qualquer um da economia, e o `!feira comprar <número>`
  compra. O anúncio não tira o item da mochila: ele só guarda item, quantidade e preço, e o
  `walletStore.buyStall` move tudo na hora da compra, numa transação só, pelo mesmo `moveItems` do
  `!vender @pessoa` (qualidade e desgaste vão junto, da pior pra melhor). Se o vendedor não tem mais
  o item, a compra falha com `errNoItems` e o `discardStall` tira o anúncio. Cada pessoa tem um
  anúncio por item (o `UNIQUE (chat, vendedor, item)`: anunciar de novo troca quantidade e preço) e
  até `stallsPerSeller`, que valem `stallSpan` (24h) e somem sozinhos (o `pitch` apaga os vencidos e
  as consultas filtram pelo `expira_em`). A comissão `stallFeePercent` (5%) é ralo: o comprador
  paga o preço cheio pro vendedor e o `debitBalance` tira a comissão dele. Anunciar passa pelo
  `heldForSale` (favorito não entra) e ilegal é recusado (vai pro `!mercadonegro` ou pro `!vender
  @pessoa`). O número do anúncio é o `id` da tabela, não o `#12` do catálogo. A lista mostra o
  vendedor pelo `DisplayName`, sem marcar ninguém. O `rivalStalls` conta os anúncios abertos dos
  outros pra sugestão da espera (`enquantoisso.go`, skill `economia-base`).
- O número de cada item (`#12`) é a posição dele no `catalog` (`item.number()`/`itemByNumber`), não
  vai pro banco. Ele aparece na `!mochila` e no `!item <nome>`, e o `!vender #12 #40 #87` (só números,
  pelo menos um com `#`, o resto pode vir sem, separados por espaço ou vírgula) vende tudo o que a pessoa tem de cada um numa transação só
  (`sellNumbered` → `sellBatch` → `walletStore.sellItems`). Item novo no meio do `catalog` empurra o número de
  todos os que vêm depois dele, então item novo vai sempre no fim do último bloco do `slices.Concat`
  do `catalog` (hoje o `gatherGoods`, em `apetrechos.go`), seja ele semente, acessório ou qualquer
  outra coisa.
- A qualidade (`qualidade.go`) é a coluna `qualidade` do `inventarios`, que faz parte da chave, então
  cada qualidade de um item é uma linha própria. Os níveis ficam em `qualities` (de `-3` horrível a `3`
  lendária, com o `0` normal), cada um com a porcentagem do `value` que vale (`price`), o selo
  (`badge`) e o peso no sorteio. A horrível e a lendária têm peso bem baixo, então só aparecem por
  sorte ou com muito sorteio extra pro lado delas. Os pesos são montados pra média dar 100%, mas o
  "melhor de N" das ferramentas depende da distribuição inteira, então mexer em porcentagem ou peso
  pede recalibrar as ferramentas.
  Só a extração sorteia qualidade (`rollQuality` no `roll`), e só pra item `gradable()`: sem efeito
  nenhum (nada de `gear`, caixa, baú, relíquia, luxo, ilegal, encanto, `trade`, `heist`...). Todo
  o resto entra como normal pelo `addItems`. O `grade` da ferramenta (`gear`, somado no `equip`) e
  o `craft.grade` da profissão dão sorteios extras de qualidade naquela extração (positivo fica com
  a melhor, negativo com a pior), e o `rotten` (hoje
  só o caçador de bússola, pra ele não lucrar com a espera curta de extração) deixa tudo com a pior
  qualidade (`worstQuality`, a primeira da lista).
  O `loadInventory` junta as linhas numa `stack` por item, com as qualidades em `grades`, então o
  resto do código continua vendo um item só; a `!mochila` mostra uma linha por qualidade. O
  `takeItems`/`takeGrades` sempre tira da pior qualidade pra melhor, e quem passa item de uma
  pessoa pra outra (`!dar`, `!vender @pessoa`, `!furtar`) usa o `moveItems` ou o
  `addGrades` pra levar a mesma qualidade junto. O `moveItems` também leva o desgaste (`usos`, lido
  pelo `wearIn` e gravado com `MAX` pelo `keepWear`), então passar ferramenta ou implante pra outra
  pessoa não zera o desgaste; quem recebe fica com o maior dos dois. O `sellItems` calcula o pagamento dentro da
  transação, com as qualidades que saíram de fato (`marketSale.percent`, não um valor pronto). Valor da mochila usa
  `stack.worth()`, nunca `quantity * value`.
- A loja de luxo (`luxo.go`) sorteia `luxuryStockSize` itens por dia e conversa (semente do dia
  em BRT e do chat) entre os `luxuryGoods`: os itens com `luxury` e os épicos/lendários com
  `keep` que a loja comum não vende. O preço é o `value` vezes `luxuryMarkup`, e o `!comprar`
  também aceita item do estoque do dia. Item com `luxury` nunca sai de caixa nem baú
  (`randomItemOf` ignora).
  - Cada item com `luxury` dá uma vantagem diferente. Os que duram têm `flex` (o `apply` mexe na
    profissão) e valem só de estar na mochila: o `walletStore.passives` lê quais a pessoa tem, em
    paralelo no `ProfessionOf`, e o `adorned` aplica todos (cada tipo conta uma vez, a quantidade
    não importa), antes do `charmed`. Os que acabam têm `bless`, o `id` de uma bênção do `charms`:
    o `!usar` gasta o item e grava a bênção em `encantos` com a própria pessoa de autor
    (`savor`, pelo mesmo `grantCharm` do `!abencoar`). Item de luxo novo precisa de um dos dois.
  - A chance de ganhar em dobro no `!trabalhar` não soma: a da profissão, a da Bolsa de grife e a
    da bênção Sorte entram com `max` no `doubleChance`, e só a Muiraquitã (relíquia) soma por
    cima, no campo próprio `relicDouble`. O sorteio e os `perks()` leem o total pelo
    `doubling()`, que devolve zero com o `noDouble` (a maldição Azar), seja qual for a ordem em que
    os encantos foram aplicados. Fonte nova de dobro usa `max`, nunca `+=`.
- O `!mercadonegro` (`mercadonegro.go`) vende os itens com `illegal` do `catalog`: armas (campo
  `arms`), drogas (campo `high`) e o contrabando (o resto, como a Balança adulterada e o Dado
  viciado, que valem pelo `flex`, o Colete à prova de balas e o Alvará de soltura falso).
  O `!usar` do alvará (`alvara.go`) tira da cadeia quem usou ou quem foi marcado, pelo `releaseHeld`
  da fiança, e marca o `crime_em` de quem usou; com chance `forgeryCaught`, o carcereiro percebe e
  a pena aumenta `escapePenalty`. Nos dois casos o alvará é gasto. O estoque do dia sai do `dailyRand` (o mesmo sorteio da
  `!luxo`, com outro sal), e cada item tem um preço do dia entre `blackMarketMinPercent` e
  `blackMarketMaxPercent` do `value`. O receptador (`!mercadonegro vender`) paga
  `blackMarketBuyback` por cento desse preço, e o `!vender` comum recusa item ilegal. Além do estoque
  do dia, a muamba (`hotGear`: ferramenta lendária com `tool`, equipamento com `relicLure` e item com
  `hot`, hoje as três ferramentas lendárias de extração, a Bússola do relicário e o Scanner da
  polícia) chega pelo `hotGearArrivals`. O item com `hot` é `unboxable()`, então só sai da muamba
  (e do `!criar`/`!milagre`). O Scanner da polícia vale pelo `flex` e soma `scannerTipoff` no
  `tipoff` da profissão, a chance de escapar do `!prender` (skill `economia-crime`). As
  ferramentas (`hotTool`) dividem um sorteio só: com `hotToolChance` vem uma delas, sorteada entre
  todas, então nunca aparecem duas no mesmo dia. O resto da muamba tem `hotGearChance` de chegar,
  cada item num sorteio próprio do `dailyRand`. Todo dia chega também um implante (item com
  `graft`, sorteado entre todos com o sal `muamba:implante`), na seção própria do `!mercadonegro`;
  ele fica fora do estoque comum mesmo quando é ilegal, e o `oneOfAKind` (muamba ou implante) é o
  que limita a uma unidade. O implante clandestino na mochila é contrabando (apreendido, comprado
  pelo receptador); o legal segue as regras da muamba. Só tem uma de cada por dia e conversa: o
  `claimHotGear` grava a compra na tabela `muambas` (`chat`, `dia`, `item`, `comprador`) na mesma
  transação da cobrança e falha com `errSoldOut` se alguém já levou; batida também conta como
  vendida, porque a mercadoria se perde. Ela não é ilegal: não é apreendida, o receptador
  não compra e a revenda é no `!vender`. Toda
  negociação passa pelo `walletStore.smuggle`: tira o que a pessoa dá, marca o `crime_em` e
  entrega a mercadoria, ou, com chance `raidOdds()` (zero pro presidente e pro deus), dá batida
  (`jailSmuggler`): cadeia por `baseJail` ajustado, `lockUp`, apreensão e solta o refém. Item
  ilegal nunca sai de caixa nem baú.
  - A arma funciona só de estar na mochila e só a mais cara conta (`bestWeapon`, lida em paralelo
    no `ProfessionOf`). O `armed` soma o `arms` na profissão: `crime` no `robBonus`,
    `theftBonus` e `kidnapBonus`, `kill` no `killChance` (então arma com `kill` libera o `!matar`
    pra qualquer profissão) e `guard` na proteção.
  - A droga é gasta no `!usar` (`takeDrug`) e guarda o efeito nas colunas `droga`/`droga_ate` da
    `carteiras`; o `standing` só devolve a droga enquanto o `droga_ate` não passou, e o `drugged`
    aplica o `high` na profissão. Só um efeito vale por vez: usar outra troca o efeito e soma
    `drugMixOverdose` à chance de overdose, que mata pelo `die`. Os campos do `high` são
    todos aplicados no `drugged` e descritos no `effects()`: `pay`, `crime` (`robBonus` e
    `theftBonus`), `escape` (`escapeBonus` e `breakoutBonus`), `heist`, `cashback`, `revive`,
    `kidnap` (`kidnapBonus`), `kill` (`killChance`, então droga com `kill` libera o `!matar`
    igual arma, e o `lethalDrugs` entra na recusa do `!matar`), `guard`, `rush` (corta o `every`
    e o `haste` em porcentagem, igual a bênção Pressa) e `luck` (`gatherLuck`). As épicas
    (Ayahuasca, Anabolizante, Pedra azul) e as lendárias (Pó de anjo, Pílula NZT) dão efeitos
    fortes, mas com overdose alta. Campo novo no `high` precisa entrar no `drugged` e no `effects()`.
  - O `walletStore.arrest` (`!prender` e `!denunciar`) apreende tudo o que é ilegal da mochila de
    quem vai preso (`seizeContraband`), e a batida também. Os outros caminhos de cadeia não
    apreendem. O que foi apreendido fica anotado na prisão, e quem tem contrato com uma organização
    do crime recebe a oferta de pagar pra ter tudo de volta (skill `economia-empresas`). O `arrest` também arranca os implantes clandestinos (`seizeImplants`, veja a skill
    `economia-dinheiro`), mas a batida não.
- Os bichinhos do `!petshop` (`petshop.go`) são itens normais do `catalog` com `shop` e `pet`: o
  `!loja` pula eles, o `!comprar` compra igual (com a mensagem de levar pra casa) e o `sources`
  aponta pro `!petshop` (`item.store()`). Bicho novo é só entrar no `catalog` com `pet: true`.
  Eles têm um efeito bem fraco no `!daily`: o `daily` lê os bichos da pessoa (`walletStore.holding`,
  o mesmo `SELECT DISTINCT` do `passives`, em paralelo com o `workTerms`) e o `fetchGift` sorteia
  cada tipo de bicho uma vez, com `fetchChance` (`petFetchChance` vezes a raridade + 1: 1% no
  comum, 2% no incomum, 3% no raro, 4% no épico). O primeiro que acerta traz um item comum
  (`petGifts`: comum, sem bicho e sem nada `unboxable()`), no máximo um por dia, e o `claim` põe
  ele na mochila na mesma transação das moedas, com qualidade normal. O texto do efeito fica no
  `petPerk`, dentro do `effectPerk`, então aparece no `!petshop` e no `!item`.
- Item com `shop` e `rotating` é oferta do dia (`rotatingStock` em `loja.go`): a `!loja` tira ele da
  lista fixa e mostra só os `rotatingShopSize` sorteados do dia na seção "⏳ Só hoje" (o
  `dailyRand` com o sal `loja`, trocando à meia-noite de BRT), e o `!comprar` recusa quando ele não
  tá no sorteio de hoje. Uma das vagas é sempre de ferramenta de extração (`featuredTool`, hoje as
  três épicas: Vara de titânio, Picareta de nióbio e Rede de náilon), em rodízio pelo dia
  (`localDay`), então cada uma volta em até 3 dias; as outras vagas são sorteadas e podem trazer
  outra ferramenta de bônus. Fora isso é item de loja normal: sai de caixa, paga o decreto `loja` e
  tem o desconto da profissão. Item de loja com `days` (dias da semana, em BRT) só aparece no "Só
  hoje" nesses dias (`weekdayStock`), fora das vagas sorteadas, e o `!comprar` recusa nos outros
  (`onSale`); hoje só a Tele Sena, aos domingos.
- Item com `carry` tem um limite por mochila (hoje só o Alarme, com `alarmCarry` = 3, que também é oferta do dia com `rotating`).
  O Alarme também tem `storeOnly`, que entra no `unboxable()`: ele só sai do "⏳ Só hoje" da `!loja`
  (e do `!criar`/`!milagre`), nunca de caixa, baú, evento nem dos achados do `!trabalhar`. Quem a
  pessoa escolhe passar recusa antes de mexer em qualquer coisa: o `roomFor` devolve `errBagFull` no
  `buyItem` (`!comprar` e `!luxo`), no `moveItems` (`!dar` e `!vender @pessoa`) e no `shiftItems`
  quando tira do imóvel, e cada comando responde com o `carryRefusal`. O resto (caixa, evento,
  achado do `!trabalhar`, `!criar`, `!milagre`) passa pelo `trimCarry` no `addGradesTo`, que tira o
  que sobrou do limite. No `!furtar`, ladrão que já tá no limite tira o item da vítima e joga fora
  (`dumped`). Quem já tinha mais que o limite antes dele existir fica com o que tem até o próximo
  item entrar.
- O `resets` (zerar esperas no `!usar`) sempre vem com um `limit` (`limiter` em `itens.go`), que é
  a espera própria entre usos e os textos dela: a `caffeine` (coluna `cafeina_em`) do café, do
  energético e do chimarrão, e o `disguise` (coluna `disfarce_em`) do Kit disfarce, que zera o
  `!roubar`, o `!furtar` e o `!sequestrar`. Item novo de zerar espera escolhe um dos dois ou cria
  outro `limiter`, com a coluna numa migração e no `resetCooldowns`.
- A horta (`horta.go`, `plantar.go`, `colher.go`, tabela `plantacoes` com uma linha por canteiro
  ocupado, guardando a `semente` e o `pronta_em`) planta item com `seed` (`crop`: o `yields` é o id
  do que nasce, o `grows` é o tempo, o `least`/`most` é quantas unidades sai por canteiro e o `wild`
  é o peso no achado de quebra). As sementes e o que nasce delas ficam no `gardenGoods`, depois dos
  implantes, e semente nova entra no fim do último bloco do `catalog` (veja o número dos itens),
  porque o `gardenGoods` já não é mais o último. O que nasce pode ser item que já existia: a Rosa e o Girassol saem da Muda de roseira e da
  Semente de girassol e continuam no tipo 💝 Presentes. O resto cai no tipo 🌱 Horta (`fromGarden`,
  logo depois de Comes e bebes), junto das sementes.
  - Todo mundo começa com `gardenBeds` canteiros, o `farmBeds` da profissão soma mais
    (`gardenRoom`) e o `!horta expandir` (`walletStore.expandGarden`) compra um de cada vez, guardado
    na coluna `canteiros_comprados` da `carteiras`. O preço de cada compra sai do `bedPrices` (o
    tamanho dele é o máximo que dá pra comprar), é um ralo (as moedas somem) e não depende do imóvel
    de propósito, pra não deixar o imóvel mais forte. O `plot` lê de uma vez a profissão, os
    canteiros, os comprados e a mochila, e o `plot.room()` é o total. O
    `!plantar <semente> [quantidade|tudo]` aceita o nome do que nasce (o `plantable` troca o item
    pela semente com o `seedFor`), tira as sementes da mochila e grava o `pronta_em` na hora, com o
    `growth` (o `farmFaster` da profissão) já aplicado, então trocar de profissão depois não mexe no
    que já tá plantado. O `walletStore.sow` recusa com `errGardenFull` quando não cabe. O `!colher`
    (`walletStore.reap`) apaga os canteiros prontos e sorteia, por canteiro, de `least` a `most`
    unidades, cada uma com a qualidade dela (`rollQuality` com o `farmGrade` da profissão). Planta
    pronta nunca estraga. O `!plantar` e o `!colher` têm `BarredInJail`, mas a horta continua
    crescendo na cadeia. O `!horta [@pessoa]` mostra os canteiros e as sementes da mochila. Hoje o
    fazendeiro tem `farmBeds` e `farmFaster`, e o agrônomo tem `farmFaster` e `farmGrade`.
  - Semente não tem qualidade (o `gradable()` olha o `seed`), tem efeito no `effectPerk` (`seedPerk`,
    então nunca sai no `!vender inuteis` nem no `bugigangas`) e resumo próprio na `!loja`
    (`seedBrief`). As comuns até raras são `shop`, e as épicas e lendárias (cacau, jabuticaba e
    pau-brasil) só saem de quebra nas extrações ou de caixa. O achado de quebra fica fora do `loot`:
    o `stray` da atividade, com a chance `seeds` dela (hoje 10% no `!catar`, 6% no `!vasculhar` e 3%
    no `!pescar` e no `!minerar`), sorteia uma semente pelo `wild` e põe na mochila junto do achado,
    então não mexe na calibração das ferramentas. O `wildSeedSources` mostra a chance de cada semente
    em cada extração no "Onde conseguir".
