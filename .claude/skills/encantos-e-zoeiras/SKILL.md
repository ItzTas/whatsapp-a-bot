---
name: encantos-e-zoeiras
description: "Bênçãos e maldições (encantos.go, charms, faith, !abencoar/!amaldicoar, itens com bless/hex, Dízimo/Multiplicação e o trigger encantos_no_ganho, announcePayouts), zoeiras pagas do !zoar, !reagir e as reações de status ruim reactToAilments (status.go). Carregue antes de mexer em algum desses."
user-invocable: false
---

# Encantos, zoeiras e status

Encantos, zoeiras pagas e as reações automáticas do bot.

- As zoeiras pagas (`zoar.go`) ficam em `zoeiras` (`chat`, `alvo`, `tipo`, `autor`, `ate`), uma por
  tipo e alvo, e pagar de novo soma o tempo. O `palhaco` é a reação 🤡 do `reactToAilments`, o
  `apelido` tranca o `!apelido` (`NicknameLocked`, só quem pagou muda) e a `maldicao` corta o ganho
  do `!trabalhar` pela metade. O `!zoar limpar` apaga as zoeiras da própria pessoa.
- O `!reagir @pessoa <emoji>` (`reagir.go`, modo `zoeira`) grava o emoji na tabela `reacoes`
  (`chat`, `id`/`id_alt`, `emoji`), um por pessoa por conversa, e marcar de novo troca o emoji. O
  `reactIfMarked` reage com ele em toda mensagem da pessoa. Ele é o primeiro reator (`AddReactor`)
  e, como uma mensagem só guarda uma reação do bot, quando ele reage o `reactToAilments` não roda.
- As bênçãos e maldições (`encantos.go`) vivem em código (`charms`, com `curse` separando as duas)
  e só o `id` vai pra tabela `encantos` (`chat`, `alvo`, `tipo`, `autor`, `ate`), uma por tipo e
  alvo; lançar de novo só renova o prazo (`MAX`), não soma. Nunca renomeie um `id`. O efeito é o
  `apply`, aplicado no `ProfessionOf` pelo `charmed`, por último, então vale em todo comando que lê
  a profissão. Quem lança é a profissão: o campo `faith` diz quais bênçãos e maldições ela tem, a
  `hexChance` da maldição (se falhar, `hexBackfire` de chance de cair em quem jogou) e os textos do
  ritual. As profissões de fé ficam na área `areaFe`, e o deus lança tudo, sem errar e sem espera
  (`pace`). Dá pra lançar em si mesmo (marcando a própria pessoa, inclusive o item com `hex`), e a
  espera é `charmCooldown` (coluna `encanto_em`). A `purificacao` não tem `apply`: apaga na hora as maldições da pessoa e a `maldicao` do `!zoar`, e
  se não tinha nada desfaz a transação com `errCharmIdle`, sem gastar a espera. Bênção ou maldição
  nova é só entrar no `charms` com o `apply` e na `faith` de pelo menos uma profissão.
  - Item também carrega encanto, sem profissão e sem a espera do `charmCooldown`. O `bless` (o
    `id` de uma bênção) faz o `!usar` gastar o item e abençoar quem usou (`savor`); com a
    `purificacao` (o Sal grosso), ele apaga as maldições e, sem nada pra tirar, o `savor` engole o
    `errCharmIdle` e devolve `idle`: o item é gasto do mesmo jeito. O `hex` (o `id` de uma maldição) é jogado em alguém com
    `!usar <item> @pessoa` (`hurlCurse`): gasta o item e pega com `itemHexChance`, com o mesmo
    `hexBackfire` das profissões (`landCharm`, que o `cast` também usa). O `use` de item com `hex`
    leva dois `%s` (quem joga e o alvo). Todo item com `bless` ou `hex` que não é de luxo cai no
    tipo `Bênçãos e maldições`, e o `!abencoar`/`!amaldicoar` sem marcar ninguém mostram os itens
    de cada encanto (`itemsCarrying`).
  - As bênçãos de extração mexem nas quatro extrações por um tempo, pelos campos `gather*` da
    profissão que o `forage` aplica antes do `equip`: a `maoboa` (`barrenHauls` e `gatherGrade`),
    o `faro` (`gatherLuck`, que soma no `plainLuck`, então a caixa de relíquia não conta no sorteio
    extra), o `achado` (`gatherRare`, o `boostRare` das ferramentas) e o `abrecaminho`
    (`gatherJunk`, o `trimJunk`). As três últimas duram 2 horas e saem de itens da `!loja` com
    `bless` (Figa de guiné, Vela pra São Longuinho e Banho de abre-caminho), além da `faith` do
    xamã, da benzedeira e da mãe de santo. Elas ficam fora da conta de tempo de se pagar das
    ferramentas. O preço do item é calibrado pelo ganho extra na venda em 2 horas de uma extração
    sem parar: com ferramenta base ele quase empata, e com ferramenta épica dá lucro. Então mexer
    no `loot` ou nas ferramentas pede conferir esses preços também.
  - O Espelho da Iara (relíquia, `mirrorItem`) liga o `mirror` da profissão pelo `flex`: o `canCast`
    deixa lançar qualquer bênção ou maldição (com a espera do `charmCooldown` normal) e o `hexOdds`
    fica com o maior entre a `hexChance` da profissão e o `mirrorHexChance`. Ele também devolve
    maldição: quando uma praga pega em quem tem o espelho na mochila, o `landCharm` (via
    `bouncesOffMirror`, que lê a mochila na mesma transação) joga ela em quem lançou, tanto no
    `!amaldicoar` quanto no item com `hex`. Não devolve a praga que a pessoa joga em si mesma, a que
    volta pelo `hexBackfire` nem a do deus (`pierce`), e só devolve uma vez (espelho contra espelho
    não fica rebatendo).
  - Dois encantos não têm `apply` e agem no próximo ganho de dinheiro: a maldição `Dízimo` (do
    pastor e da bruxa), com `toll` (`tithePercent`), e a bênção `Multiplicação` (do padre e do
    pastor), com `boon` (`boonPercent`) e `boonCap`. O `grantCharm` grava esses campos nas colunas
    `parte`, `bonus` e `teto` de `encantos`, e quem age é o trigger `encantos_no_ganho` da
    `carteiras` (migração `00074`), não o Go: qualquer `UPDATE` que suba o saldo de quem tem um
    desses encantos passa a `parte` do que entrou pro `autor` e credita o `bonus` (limitado ao
    `teto`) em moedas novas, tudo calculado em cima do ganho original, na mesma transação. Depois
    ele apaga os encantos usados (menos os com a coluna `continuo`, gravada pelo `lasting` do
    encanto, hoje só o Pacto do Livro de São Cipriano, que cobra em todo ganho até o `ate`; o
    trigger foi recriado na migração `00078`) e grava um aviso por encanto em `encantos_avisos`. Por
    isso vale em todo caminho de dinheiro sem gancho nenhum. Um ganho que daria valor zero não gasta
    o encanto, e os créditos do próprio trigger não disparam outro (o SQLite roda sem
    `recursive_triggers`). Os eventos com escolha também lançam esses encantos: a Multiplicação pelo
    `enchant` e o Dízimo e o Pacto pelo `grudge`, que grava a vítima do crime como `autor` (skill
    `economia-eventos`). Encanto novo desse tipo é só usar os mesmos campos; não crie outro
    trigger no `saldo`, senão um dispara o outro. O `announcePayouts` (`ganhos.go`) entra pelo
    `AddAfterMessage`, e o `HandleMessage` chama ele com `defer` em toda mensagem de conversa
    liberada, que apaga os avisos da conversa e manda no grupo, então o que cair no `runLottery` ou
    no `runTaxes` só é avisado na próxima mensagem. Quando a praga volta (`hexBackfire` ou espelho),
    o `landCharm` grava o alvo original como `autor`, então quem jogou paga pra ele; o Dízimo em si
    mesmo (`autor = alvo`) é ignorado pelo trigger. O `purges()` é só a `purificacao`.
  - A maldição `assombracao` (🏚️ Casa assombrada, `assombracao.go`, da bruxa e do exorcista)
    assombra o imóvel e dura `hauntHold` (7 dias, bem mais que as outras). Só pega em quem tem
    imóvel: o `cast` confere o alvo antes de gastar a espera (`needsHome`, falha com
    `errHomeless`), e se a praga voltar (`hexBackfire` ou espelho) pra quem não tem casa, o
    `landCharm` marca `vacant` e não grava nada. O `apply` (`haunted`) inverte o que o imóvel dá
    (proteção, `dailyBonus` e `payBonus` viram o contrário, lendo o `p.estate` que o `housed` já
    pôs) e o `taxPayers` dobra o IPTU (`iptu`, `hauntTaxFactor`). A `purificacao` não tira ela
    (`purgeCurses` pula o `haunts()`): o espírito só sai no fim do prazo ou no
    `!imovel exorcizar` (`exorcize`), que passa `hauntRansomDays` dias de IPTU do saldo de quem
    tá assombrado pro `autor` do encanto (não é ralo) e apaga a linha. O texto de como se livrar
    sai do `c.cure()`, que troca o `cureText` por esse caminho.
  - A Carranca do Velho Chico (relíquia passiva, `carrancaItem`) segura o espírito sem apagar o
    encanto: o `apply` dela (`warded`) liga o `warded` e soma `carrancaBoost` por cento do que o
    imóvel dá (proteção, `dailyBonus` e `payBonus`), e como o `adorned` roda antes do `charmed`, o
    `haunted` vê o `warded` e não inverte nada. O IPTU dobrado olha o mesmo: o `!imposto` usa o
    `hauntedHome()` e o `taxPayers` tem um `NOT EXISTS` da Carranca no `inventarios`. O
    `hauntedHomeText` do `!imovel` mostra o espírito rondando a casa sem entrar. A maldição continua
    gravada, então perder a Carranca antes do fim do prazo faz o espírito voltar a valer na hora.
    É diferente do Espelho da Iara, que só devolve a praga na hora em que alguém lança.
- O `reactToAilments` (`status.go`), o reator da economia (`AddReactor`), reage em toda mensagem de
  quem tá com algum status ruim, se o modo `economia` tiver ligado. O `walletStore.ailments` lê tudo
  numa consulta só. Como o WhatsApp só aceita uma reação do bot por mensagem, vale o primeiro da
  lista do `ailments.emoji`: morto ⚰️, preso 🔒, refém 🪢, palhaço do `!zoar` 🤡, amaldiçoado (o emoji
  da maldição do `encantos`, ou 🧿 pra `maldicao` do `!zoar`), drogado 😵‍💫 e saúde em até um terço do
  `MaxHealth` 🤕. Status novo é só entrar nessa lista, na posição da prioridade.
