---
name: economia-base
description: "Base da economia do bot (carteiras, chave canônica b.Account, profissões em profissoes.go/profissao.go, ProfessionOf/standing, deus.go, oficio.go e o !trabalhar, ralos de dinheiro, walletStore.spend, !economia resetar intervalos). Carregue antes de mexer em qualquer comando ou arquivo da economia, junto da skill economia-* do assunto."
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
  `!prender`, `!fugir`, `!libertar`, `!reviver`, `!denunciar`, `!sequestrar`, `!venderrefem`, `!assalto`, `!usar`, pensão e indenização), sempre lendo a
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
  luxo e os ilegais, sem cobrar nada.
- O `!economia resetar intervalos` (dono, em `economia.go`) zera as colunas `*_em` de espera da
  `carteiras` (`resetCooldowns`), da conversa toda ou só de quem foi marcado. Ele também zera
  `preso_ate`, `morto_ate` e `crime_em` e apaga a linha de `sequestros` de quem é refém, então
  solta da cadeia, da morte e do cativeiro e limpa a ficha. Também enche a saúde (`saude` e
  `saude_em`). Nunca mexe em saldo, mochila,
  poupança, empréstimo nem presidência. Coluna de espera nova entra na lista do `resetCooldowns`.
- Os ralos de dinheiro existem porque a economia só tinha torneira: o dinheiro gasto neles some
  (nunca vai pra ninguém), a não ser quando o texto diz o contrário. Preço de ralo, quando dá, é
  uma porcentagem do saldo com piso, pra pesar igual pra rico e pobre. Pra cobrar e fazer alguma
  coisa na mesma transação, use `walletStore.spend` (debita com `debitBalance` e chama o resto).
- O `ProfessionOf` lê tudo de uma vez (`walletStore.standing`) e aplica, depois do
  `presidential()`, o imóvel (`housed`), o segurança do `!seguranca` (`escorted`, coluna
  `seguranca_ate`) e a maldição do `!zoar` (`hexed`). Efeito novo que mude `guard`, ganho ou
  bônus de alguém por fora da profissão entra nesse mesmo ponto.
