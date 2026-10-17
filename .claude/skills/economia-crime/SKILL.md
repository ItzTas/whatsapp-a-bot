---
name: economia-crime
description: "Crime e justiça da economia (!prender, !fugir, !libertar, !denunciar, !sequestrar e !venderrefem, !matar, !recompensa, !reviver, morte bury/die, saúde saude.go e !hospital, !assalto, !quadrilha, abuso de poder abuseKinds, fiança/hospital/resgate buyExit, !subornar, tribunal do !processar/!processaria, baseJail/lockUp/sentence, crime_em). Carregue antes de mexer em qualquer um desses comandos ou num caminho novo de cadeia, morte ou flagrante."
user-invocable: false
---

# Economia: crime, cadeia e morte

Cadeia, flagrante, morte, saúde e tribunal.

- O `!processaria` (`processaria.go`) e o `!processar` (`processar.go`) dividem o mesmo tribunal:
  o processo aberto fica em `TrialChats` (um por conversa, pros dois comandos) e a enquete do júri
  em `trialPolls`, e depois de 2 minutos o veredito sai pela contagem da enquete.
  - O `!processaria` é só zoeira de IA: a IA escreve o julgamento e as duas sentenças num JSON só,
    sem moedas, sem profissão e sem espera (`trialSides` vazio).
  - O `!processar` é da economia e só abre contra quem tem uma queixa na tabela `queixas`
    (`vitima`, `acusado`, `motivo`, `em`) ou deve pra quem acusa no `!agiota` com o prazo vencido.
    O `!denunciar` e o `!prender` gravam a queixa (`fileGrievance`) quando o alvo tava com a ficha
    limpa, e ela vale por `grievanceWindow`. O texto do julgamento é fixo (`grievanceKinds`, um por
    `motivo`, ou `debt.script` pro calote) e a queixa é apagada quando o processo abre, então cada
    queixa rende um processo só; com queixa e dívida ao mesmo tempo, a queixa vem primeiro. A
    dívida não é apagada: no calote (`trialCase.debt`), a condenação troca a indenização por
    `courtDebtCollection`, que paga a dívida com o que o devedor tiver no saldo (`repayDebt`). Quem acusa espera `trialEvery` entre
    processos (coluna `processo_em` da `carteiras`), carimbada na abertura e zerada se ela falhar.
    Na sentença, a indenização (ou as custas de quem perdeu o processo) é transferida, e a
    profissão de cada lado (`trialSides`, lida na abertura) entra no jogo: `trialEdge` soma votos
    pro próprio lado (negativo soma pro outro, a "ficha suja" do crime) e `trialEvery` encurta a
    espera.
- O `!prender` (`prender.go`) só funciona pra profissão com `jailTime` (policial, delegado, juiz), e
  só em quem tentou `!roubar`, `!furtar` ou `!sequestrar` nas últimas `crimeWindow`: a tentativa
  marca o `crime_em` na `carteiras` (`rob`, `stealItem` e `kidnap`, via `markCrime`) e a prisão zera
  ele, então cada crime rende uma prisão só. Toda ida pra cadeia parte do mesmo `baseJail` (o
  `jailTime` das profissões que prendem também é ele), e o `jailFactor` da profissão de quem é preso
  e o decreto de pena somam ou tiram uma porcentagem desse tempo (`sentence`); caminho novo de
  cadeia usa o `baseJail` e não um tempo próprio; o `arrestEvery` é a espera entre prisões (coluna
  `prisao_em`). A saída da cadeia fica em `preso_ate` na `carteiras`. Toda ida pra cadeia
  (`!prender`, `!denunciar`, X-9, pego no `!matar`, no `!sequestrar`, no `!assalto` e no desvio da
  `!igreja`) cobra uma
  multa de `fineMinPercent` a `fineMaxPercent` por cento do saldo e corta `arrestEaseCut` por cento
  da espera do `!fugir` que faltava (`lockUp`, via `easeEscape`), e as moedas somem, menos no
  `!prender`, em que o `arrest` (com `copKeepsFine`) credita a multa em quem mandou prender; a
  `!fianca` é outra coisa, é o preço de sair antes. Caminho novo que mande alguém pra cadeia precisa
  chamar o `lockUp` também. Comando com `BarredInJail` no `economyCommands` é barrado pela guarda da
  economia (`economyGuard`) enquanto quem mandou tá preso ou em cativeiro (hoje `!apostar`,
  `!cassino`, `!blackjack`, `!crash`, `!corrida`, `!roubar`, `!furtar`, `!sequestrar`,
  `!venderrefem`, `!matar`, `!assalto`, `!quadrilha` e `!libertar`).
- O `!fugir` (`fugir.go`) tenta tirar quem tá preso da cadeia, com chance `escapeChance` mais o
  `escapeBonus` da profissão (negativo pra polícia), uma tentativa a cada `escapeCooldown`
  (coluna `fuga_em`). Fugir zera o `preso_ate` e marca o `crime_em`, então a polícia pode
  prender de novo; falhar soma `escapePenalty` ao `preso_ate`. Quem não tá preso mas tá em
  cativeiro (vendido ou não) tenta fugir do sequestro, com chance `breakoutChance` mais o
  `breakoutBonus` da profissão (positivo pra polícia, segurança, mágico, chaveiro...), na mesma
  espera `fuga_em`. Dar certo apaga a linha de `sequestros` sem marcar crime; falhar soma
  `breakoutPenalty` ao `ate`. Se a pessoa tá presa e em cativeiro ao mesmo tempo, a cadeia vem
  primeiro.
  - O `!fugir @pessoa` (`helpEscape`/`walletStore.rescue`) é tentar tirar outra pessoa, com as
    chances da profissão de quem ajuda e a espera `fuga_em` dela; quem tá preso ou em cativeiro
    não ajuda ninguém. Na cadeia, dar certo zera o `preso_ate` e marca o `crime_em` dos dois
    (`bustOut`); dar errado marca o `crime_em` de quem ajudou e tem `rescueCaught` de chance de ele
    ir preso na hora, por `baseJail` ajustado pelo `jailFactor`, com `lockUp` (zero pra quem
    `slipsAway()`). No cativeiro, dar certo apaga a linha de `sequestros` e falhar não tem castigo.
- O `!libertar` (`libertar.go`) é o `!fugir` feito por outra pessoa: só funciona pra profissão
  com `releaseChance` (juiz, advogado, delegado), em quem tá preso ou em cativeiro (a cadeia vem
  primeiro), uma tentativa a cada `releaseCooldown` (coluna `libertacao_em`). Dar certo zera o
  `preso_ate` ou apaga a linha de `sequestros`, sem marcar crime em ninguém. Falhar tem
  `releaseFineChance` de cobrar uma multa entre `releaseFineMin` e `releaseFineMax` por cento do
  saldo de quem tentou, e as moedas somem (não vão pra ninguém). O presidente paga a multa com
  `presidentFineDiscount` de desconto (`releaseFineShare` da profissão).
- O `!denunciar` (`denunciar.go`) é o `!prender` de quem não é da polícia: usa o mesmo
  `walletStore.arrest` (mesma checagem de `crime_em`, mesmo resgate de refém), mas com chance
  `reportChance` mais o `reportBonus` de quem denuncia, cadeia `baseJail` e espera
  `reportCooldown` (coluna `denuncia_em`). O `!prender` passa chance 1 pro `arrest`. Profissão
  com `noSnitch` (bandido) não denuncia: quem tenta vai preso na hora.
- O `!sequestrar` (`sequestrar.go`) guarda o refém na tabela `sequestros` (`refem`/`refem_jid`,
  `sequestrador` e `ate`), um refém por sequestrador, por `hostageHold`; linha vencida é
  ignorada e apagada na próxima tentativa. A chance é `kidnapChance` mais o `kidnapBonus` de quem
  sequestra menos o `guard` da vítima, e a espera é `kidnapCooldown` (coluna `sequestro_em`, ou o
  `kidnapEvery` da profissão). A tentativa marca o `crime_em`; se falhar, paga indenização e tem
  `kidnapCaughtChance` de ir preso na hora (mais `copVictimCaught` se a vítima tem `jailTime`,
  por `baseJail` ajustado pelo `jailFactor`). O `!prender` em quem sequestrou solta o refém
  (`freeHostage`). O `!venderrefem` abre uma negociação efêmera em `hostageSales` (uma por
  sequestrador) que só fecha depois de `hostageSaleWait`; até lá o `!prender` ou o `!fugir` melam o
  negócio, e o preço e o tempo que sobra são lidos no fechamento. Na venda, paga, com moedas
  novas, o ganho da profissão do refém vezes `hostagePriceFactor`, mais o `ransomBonus` de quem
  vende, e troca o `sequestrador` por `NULL` e corta o `ate` pra no máximo `soldHostageHold` a
  partir da venda: o refém continua barrado até lá, mas a polícia não solta mais.
- O `!matar` (`matar.go`) só funciona pra profissão com `killChance` (bandido, açougueiro,
  assaltante de banco, sequestrador e matador de aluguel). A chance é o `killChance` menos o
  `guard` da vítima, e a espera é `killCooldown` (coluna `assassinato_em`). A tentativa marca o
  `crime_em`, então dá pra `!prender` e `!denunciar`. Se ia dar certo e a vítima tem o Colete à
  prova de balas (`vestItemID`, ilegal), o `kill` gasta um colete (`spendItem`) e ela não morre,
  menos contra deus e contra quem tem a Foice da Comadre Morte (`killOdds.pierce`). Se der certo,
  a vítima morre pelo `bury` (`saude.go`, veja a saúde abaixo) com o `funeral` do `killOdds`: o
  normal (`plainFuneral`) ou, com a Foice (`reaper`, que também soma `reaperKillBonus` no
  `killChance` e libera o `!matar` pra qualquer um), `reaperHoldFactor` vezes o `deathHold` e o
  enterro de `reapLossMinPercent` a `reapLossMaxPercent` por cento creditado no matador (`heir`);
  enquanto tá morta, o
  `economyGuard` barra qualquer comando da categoria de economia (`holdIfDead`), não só os com
  `BarredInJail`. Se der errado, tem `killCaughtChance` de ir preso na hora (mais
  `copVictimCaught` se a vítima tem `jailTime`), por `baseJail` ajustado pelo `jailFactor`.
- A `!recompensa @pessoa <valor>` (`recompensa.go`) desconta o valor de quem bota na hora e guarda
  na tabela `recompensas` (`chat`, `alvo`, `autor`, `valor`, `em`), uma linha por autor e alvo
  (botar de novo soma). Quando o `!matar` dá certo, o `kill` chama o `claimBounty` na mesma
  transação: soma tudo que tem na cabeça da vítima, apaga as linhas e credita no matador, então
  vale também pra cobrança do `!agiota` e pro deus. A `!recompensa retirar @pessoa` devolve só a
  parte de quem mandou. Nenhum outro jeito de morrer (overdose, saúde zerada) paga a recompensa.
- O `!reviver` (`reviver.go`) tira a morte antes do prazo: só funciona pra profissão com
  `reviveChance` (necromante) ou pro presidente (`presidentRevive` mínimo), em quem tem `morto_ate`
  no futuro, uma tentativa a cada `reviveCooldown` (coluna `ressurreicao_em`). Dar certo zera o
  `morto_ate`; falhar só gasta a espera. O Livro de São Cipriano (relíquia, `ciprianoItem`) dá
  `reviveChance` 1 e liga o `soulPact`: dar certo grava o `soulPactCharm` (o Pacto, maldição com
  `toll` de `soulPactPercent` e `lasting`) no revivido, com quem reviveu de `autor`, por
  `soulPactHold`. O Pacto fica fora do `charms` (ninguém lança ele, nem o Espelho da Iara), mas o
  `charmByID` acha ele e a `purificacao` apaga ele junto das maldições.
- Toda morte passa pelo `bury` (`saude.go`), e o `die` é ele com o `plainFuneral`: grava
  `morto_ate` pelo `hold` do `funeral` (metade pra quem tem o Livro de São Cipriano na mochila,
  devolvido em `death.hold` pros textos), volta a saúde pra
  `MaxHealth`, solta o refém que a pessoa tiver (`freeHostage`) e o enterro queima de
  `deathLossMinPercent` a `deathLossMaxPercent` por cento do saldo (`burnBalance`, sem piso, pra
  não zerar quem tem pouco), e as moedas somem, a não ser que o `funeral` tenha `heir` (a Foice). Hoje chamam ele o `kill` do `!matar`, a overdose do
  `takeDrug` e o `hurt` da saúde zerada. Caminho novo de morte usa o `die` e mostra o `funeralText`.
- A saúde (`saude.go`) fica nas colunas `saude`/`saude_em` da `carteiras`: vale o número guardado
  mais 1 ponto a cada `healthRegenEvery` desde o `saude_em`, até `MaxHealth`, calculado na leitura
  (`readHealth`), sem nada rodando em segundo plano. Quem muda a saúde grava os dois juntos
  (`setHealth`). O `!trabalhar` das profissões com `hazard` (as perigosas de obra, transporte,
  cozinha, campo, polícia, crime...) e as quatro extrações (o `hazard` de cada `activity`)
  sorteiam um ferimento (`hazard.roll`, dano de `harm/2` a `harm`), e o `hurt` aplica na mesma
  transação, só quando o comando saiu de fato (com espera pendente ninguém se machuca). A chance
  cai pela porcentagem de `safety`, até `maxSafety`: no trampo, o `safety` da profissão (o Kit EPI,
  pelo `flex`); nas extrações, o `safety` do `gear` que cabe nela (capacete, luva de borracha,
  boia salva-vidas e roupa de apicultor), somado no `equip`. Saúde zerada mata pelo `die`. Item
  com `heal` (Curativo e Kit de primeiros socorros, tipo `Saúde`) é gasto no `!usar` e cura na
  hora (`walletStore.heal`, que recusa com a saúde cheia sem gastar), e o `!hospital` de quem tá
  vivo e machucado (`treat`) enche a saúde cobrando o `hospitalPrice` vezes a fração que falta
  (`carePrice`). O `!saude` mostra a saúde e a chance de se machucar em cada lugar, já com a
  proteção. Profissão perigosa nova é só ganhar um `hazard`; os `wounds` entram logo depois do
  `@pessoa` (tipo "caiu do andaime"), então começam com verbo no passado.
- O `!assalto` (`assalto.go`) junta uma equipe de `heistSize` pessoas em `heistCrews` (uma por
  conversa, efêmera): quem manda primeiro abre, e se a equipe não fechar em `heistWindow` o
  assalto é cancelado. A espera é só checada ao entrar (`cooldownLeft`) e carimbada pra todo mundo
  na hora do assalto (coluna `assalto_em`), junto do `crime_em`, então a polícia pode `!prender`
  quem tava lá. A chance é `heistChance` mais o `heistBonus` da profissão de cada um mais o
  `heist` de cada item de equipamento (`catalog` em `itens.go`) achado na mochila de alguém da
  equipe; cada tipo de item conta uma vez e uma unidade é gasta. Se der certo, a equipe divide
  moedas novas entre `heistLootMin` e `heistLootMax`; se der errado, cada um tenta escapar com o
  `escapeOdds` da profissão, e quem não escapa vai pra cadeia por `baseJail` ajustado pelo
  `jailFactor` (via `sentence`), com o `crime_em` zerado.
- A `!quadrilha @pessoa` (`quadrilha.go`) é o `!roubar` em grupo: junta até `gangSize` pessoas em
  `gangs` (uma por conversa, efêmera e salva com `KeepGame`) contra uma vítima só, marcada por quem
  abre; quem entra depois manda `!quadrilha` (marcar outra pessoa é recusado). Com `gangSize` sai
  na hora; se a `gangWindow` acabar com pelo menos `gangMin`, sai assim mesmo, e com menos é
  desfeita. A espera é a do `!roubar` (`robCooldown()` de cada um), checada ao entrar e carimbada
  pra todo mundo na hora do roubo, junto do `crime_em`. A chance é `robSuccessChance` mais o
  `robBonus` de cada um mais `gangHelp` por pessoa a mais, menos o `guard` da vítima e o toque de
  recolher. O alarme da vítima protege igual ao `!roubar`. Se der certo, a quadrilha leva de
  `gangLootMin` a `gangLootMax` por cento do saldo da vítima, dividido igual, e a parte de cada um cresce o `robHaul` da profissão dele (hoje só o Chapéu de Lampião, o mesmo que aumenta o que leva no `!roubar`); se der errado, cada
  um paga de 10% a 20% do próprio saldo pra vítima. Ninguém vai preso na hora.
- O abuso de poder (`abuseKinds` em `prender.go`) deixa usar um comando mesmo com a espera dele
  pendente. Hoje tem quatro tipos: prisão (`!prender`), trabalho (`!trabalhar`), roubo (`!roubar`) e
  sequestro (`!sequestrar`). Cada `abuseKind` tem a coluna de espera dele na `carteiras` (`abuso_em`,
  `abuso_trabalho_em`...) e a coluna de cargas na `presidencias` (`abusos`, `abusos_trabalho`...).
  Quem pode abusar de quê é o campo `abuses` da profissão (delegado e juiz: prisão; gerente e mestre
  de obras: trabalho; assaltante de banco: roubo; sequestrador: sequestro), e o presidente pode
  todos. O comando pede um `abuser` com `job.abuse(kind)` e passa pro `stampOrAbuse`: se a espera
  ainda não passou, o presidente gasta primeiro uma das `presidentAbuses` cargas do mandato daquele
  tipo e depois a profissão gasta a espera do abuso (`kind.every`). Só é gasto quando o comando ia
  sair de fato. Tipo novo precisa de entrada no `abuseKinds`, das duas colunas numa migração, da
  coluna de espera no `resetCooldowns` e do `stampOrAbuse` no comando, com o `note` na resposta.
- Fiança, hospital e resgate (`fianca.go`, `hospital.go`, `resgate.go`) passam pelo `buyExit`: o
  preço sai do saldo de quem tá preso/morto/refém e quem paga pode ser outra pessoa. A saída é um
  `UPDATE`/`DELETE` condicionado ao prazo (`releaseHeld`), e se a pessoa já saiu ele devolve
  `errNotHeld` e a transação desfaz a cobrança. O resgate apaga a linha de `sequestros`, o que
  mela uma venda de refém pendente. O `!hospital` tem `WhileDead` no `economyCommands`, que é
  o que deixa ele passar pelo `holdIfDead`; com a pessoa viva, ele não passa pelo `buyExit` e vira o
  tratamento da saúde (`treat`).
- O `!subornar pagar` (`suborno.go`) cobra `bribePercent` do saldo, com piso `bribeFloor`, e as
  moedas somem. Em troca grava `propina_ate` na `carteiras` por `bribeHold` (pagar de novo soma o
  tempo), e o `paidOff` marca `bribed` na profissão. Enquanto vale, o `untouchable()` (que junta
  presidente e subornado) barra a cadeia como a imunidade do presidente: o `arrestShield` recusa o
  `!prender` e o `!denunciar`, a chance de flagrante no `!sequestrar` e no `!matar` e a batida do
  `!mercadonegro` viram zero e o `jailCrew` do `!assalto` pula a pessoa. O X-9 do `noSnitch`
  continua valendo, porque quem entrega é a quadrilha. O flagrante na hora (`!sequestrar`, `!matar`,
  `!fugir @pessoa` e o `jailCrew` do `!assalto`) checa o `slipsAway()`, que junta o `untouchable()`
  com o `elusive` do Gorro do Saci; o Gorro só escapa do flagrante, não do `!prender`, do
  `!denunciar` nem da batida do `!mercadonegro`. No `!processar`, o `trialBribes` lê a
  propina dos dois lados na hora da sentença: se só um pagou, o `riggedVerdict` dá a vitória pra
  ele sem olhar a enquete (nem o quórum); se os dois pagaram, o júri decide normalmente.
