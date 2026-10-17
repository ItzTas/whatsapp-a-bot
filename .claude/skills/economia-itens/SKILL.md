---
name: economia-itens
description: "Itens da economia (catalog em itens.go, inventarios, raridades, relíquias e míticos, qualidade.go, ferramentas de extração e a calibração do tempo de se pagar, equip/loot/roll, tipos itemCategories, números #12, vendas em lote do !vender, !loja e ofertas do dia, !luxo, !mercadonegro, !petshop, resets/limiter do !usar). Carregue antes de mexer em item, loja, mochila, extração (!pescar, !minerar, !vasculhar, !catar) ou venda."
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
  Comadre Morte e Livro de São Cipriano) fica fora da loja, das atividades, da `!luxo` e das caixas e baús comuns. Ele só sai
  de caixa com `relicOdds` (hoje o Relicário perdido), que no `roll` troca o prêmio por uma
  relíquia aleatória nessa chance, do `!leilao` nos dias de relíquia e do `!criar` e do `!milagre` de deus; depois passa de mão em
  mão, mas nunca pelo `!furtar` (o `theftChoice` tira as relíquias do sorteio e o furto mirado
  numa relíquia é recusado). A caixa com `relicOdds` também não sai de outra caixa e só cai nas extrações (`loot` das
  `activities`, peso 2). Equipamento com `relicLure` (hoje a Bússola do relicário, que sai de
  caixa comum e, com peso 1, das extrações) multiplica esse peso no `equip` (`lureRelics`) e, com `spent`, é gasto na próxima
  extração. O filtro de metal do Ímã tira a caixa de relíquia do sorteio, mesmo com a bússola. O
  `randomItemOf` pula tudo que é `unboxable()` (luxo, ilegal, relíquia, mítico e
  caixa de relíquia). Ferramentas (`gear` com `tool`) não somam entre si: se a pessoa tem mais de
  uma da mesma atividade, o `equip` só aplica a de maior `value` (`bestTool`), e as outras ficam
  paradas na mochila. Os acessórios (`gear` sem `tool`, como isca, capacete e bússola) continuam
  valendo junto com ela. A sorte de acessório com `skipsRelics` (a isca) vai pro `plainLuck` do
  `roll`, e nesses sorteios extras a caixa de relíquia é ignorada. A Excalibur é arma legal: o `bestWeapon` lê toda arma do `catalog` (não só as
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
  a cada comando, o efeito liga e desliga sozinho, sem estado guardado.
- Mítico é outra raridade própria (`mythic`, entre a `legendary` e a `relic`): são itens de
  extração que só servem pra vender (sem `use`, sem `keep`, sem efeito), dois por atividade, com
  peso 1 no `loot`. Eles só saem com relíquia: o `equip` tira os míticos do `loot` quando nenhum
  item equipado naquela extração é relíquia, então a ferramenta relíquia da atividade (ou o Saco
  do Homem do Saco) e a Vela do Negrinho do Pastoreio liberam eles, e o `source` da atividade
  mostra quais relíquias liberam no "Onde conseguir". São `unboxable()`, então não saem de caixa
  nem de baú, mas têm qualidade e podem ser furtados. Na venda pro mercado, o `marketPercentOf`
  paga pelo menos `mythicSalePercent` (hoje 90%) por eles, ou o `salePercent` da profissão se for
  maior; ele é aplicado dentro do `marketPayoutOf` e do `marketPayout`, então vale no `!vender`
  (todas as formas), no valor de mercado da `!mochila` e no `!item`.
  O `!vender inuteis` vende mítico junto com o resto.
- As ferramentas de extração que não são relíquia (rara, épica e lendária de `!pescar`,
  `!minerar` e `!catar`) têm um tempo padrão de se pagar, hoje **135 minutos**, contando o que a
  pessoa recebe vendendo no `!vender`, não o valor cheio dos itens. O tempo é o `value` da
  ferramenta dividido pelo líquido por hora: o valor médio do melhor item entre `luck + 1`
  sorteios do `loot` montado pelo `equip` (item `gradable()` vezes a qualidade média da melhor de
  `grade + 1` qualidades), vezes o `marketPercentOf` do item (o `marketSalePercent`, ou o
  `mythicSalePercent` nos míticos, sem o `salePercent` de profissão), menos `breaks` vezes o `value` cheio da ferramenta (a reposição sai pelo preço da
  loja), vezes os usos por hora da espera. O ajuste é nos atributos do `gear` (`luck`,
  `fewerJunk`, `moreRare`), não no preço nem na espera: o `faster` é o mesmo pra toda ferramenta
  de extração da mesma raridade, hoje 0 na rara (10 minutos, igual à base), 40 na épica (6), 50
  na lendária (5) e 80 na relíquia (2). O `grade` também é fixo por raridade: 1 na rara, 2 na
  épica, 3 na lendária e 4 na relíquia. Ferramenta nova ou mudança no `loot` de uma
  atividade precisa recalibrar as ferramentas dela pra esse tempo. As ferramentas base (Vara de
  pesca, Picareta e Rede de insetos) ficam de fora e se pagam em uns 60 minutos. Como o `loot`
  da mina vale mais, as picaretas têm um `luck` a menos que a vara e a rede da mesma raridade.
  As relíquias de extração que fazem o papel da ferramenta (Vara de Iemanjá, Picareta de Serra
  Pelada, Rede do Curupira e o Saco do Homem do Saco, que é acessório porque o `!vasculhar` não
  tem ferramenta base) seguem a mesma conta com outro tempo, hoje **30 minutos**, usando o `value`
  delas como preço; como nunca quebram, não têm custo de quebra. Os míticos entram nessa conta
  (só elas acham), então mexer em mítico também pede recalibrar as relíquias de extração. A Vela
  do Negrinho do Pastoreio fica de fora, porque só soma em cima da ferramenta que a pessoa já tem.
- O tipo de cada item (`itemCategories` em `itens.go`) não é um campo: é a primeira regra da lista
  que bate com os campos do item (`pet`, `arms`, `heist`, `gift`, o loot das `activities`...), e o
  que não bate em nada vira `Raridades`. O `Contrabando` pega o ilegal que não é arma nem droga, e
  o `Vantagens` pega os itens com `flex` que não são relíquia, luxo nem ilegais. O `!item`, o `!item <tipo>`, a `!mochila` e a `!loja`
  agrupam por ele (`categoryGroups`), com a raridade na bolinha da linha. Item novo cai no tipo
  sozinho; tipo novo é só entrar na lista, na posição certa, porque a ordem decide o empate.
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
- O número de cada item (`#12`) é a posição dele no `catalog` (`item.number()`/`itemByNumber`), não
  vai pro banco. Ele aparece na `!mochila` e no `!item <nome>`, e o `!vender #12 #40 #87` (só números,
  todos com `#`, separados por espaço ou vírgula) vende tudo o que a pessoa tem de cada um numa transação só
  (`sellNumbered` → `sellBatch` → `walletStore.sellItems`). Item novo no meio do `catalog` empurra o número de
  todos os que vêm depois dele.
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
  `addGrades` pra levar a mesma qualidade junto. O `sellItems` calcula o pagamento dentro da
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
  do dia, a muamba (`hotGear`: ferramenta lendária com `tool` e equipamento com `relicLure`, hoje as
  três ferramentas lendárias de extração e a Bússola do relicário) chega pelo `hotGearArrivals`. As
  ferramentas (`hotTool`) dividem um sorteio só: com `hotToolChance` vem uma delas, sorteada entre
  todas, então nunca aparecem duas no mesmo dia. O resto da muamba tem `hotGearChance` de chegar,
  cada item num sorteio próprio do `dailyRand`. Só tem uma de cada por dia e conversa: o
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
    `drugMixOverdose` à chance de overdose, que mata pelo `die`.
  - O `walletStore.arrest` (`!prender` e `!denunciar`) apreende tudo o que é ilegal da mochila de
    quem vai preso (`seizeContraband`), e a batida também. Os outros caminhos de cadeia não
    apreendem.
- Os bichinhos do `!petshop` (`petshop.go`) são itens normais do `catalog` com `shop` e `pet`: o
  `!loja` pula eles, o `!comprar` compra igual (com a mensagem de levar pra casa) e o `sources`
  aponta pro `!petshop` (`item.store()`). Bicho novo é só entrar no `catalog` com `pet: true`.
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
- O `resets` (zerar esperas no `!usar`) sempre vem com um `limit` (`limiter` em `itens.go`), que é
  a espera própria entre usos e os textos dela: a `caffeine` (coluna `cafeina_em`) do café, do
  energético e do chimarrão, e o `disguise` (coluna `disfarce_em`) do Kit disfarce, que zera o
  `!roubar`, o `!furtar` e o `!sequestrar`. Item novo de zerar espera escolhe um dos dois ou cria
  outro `limiter`, com a coluna numa migração e no `resetCooldowns`.
