---
name: economia-crime
description: "Crime e justiça da economia (!prender, !fugir, !libertar, !denunciar, !sequestrar e !venderrefem, !matar, !recompensa e o procurado do dia runWanted, !reviver, morte bury/die, saúde saude.go e !hospital, !assalto, !quadrilha, abuso de poder abuseKinds, fiança/hospital/resgate buyExit, flagrante flagrantOdds, tribunal do !processar/!processaria, baseJail/lockUp/sentence, crime_em). Carregue antes de mexer em qualquer um desses comandos ou num caminho novo de cadeia, morte ou flagrante."
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
  `prisao_em`). O `tipoff` de quem vai preso (hoje só o Scanner da polícia, muamba do
  `!mercadonegro`) tira chance do `arrestOdds` do `!prender` (inclusive o do calote do `!agiota`):
  quando ele acerta, o `arrest` volta com `dismissed`, a espera de quem prendeu já foi gasta e o
  `crime_em` continua, então dá pra tentar de novo. Do mesmo jeito, o `slip` do `arrestOdds` (o
  `!prender` passa `jackpotArrestSlip`) dá pra quem é alvo fácil com prêmio fresco
  (`slipsArrest`, veja abaixo) uma chance de o advogado soltar na hora: o `arrest` volta com
  `lawyered`, com a espera gasta e o `crime_em` intacto. O `!denunciar` não olha o `tipoff` nem o
  `slip`. A saída da cadeia fica em `preso_ate` na `carteiras`. Toda ida pra cadeia
  (`!prender`, `!denunciar`, X-9, pego no `!matar`, no `!sequestrar`, no `!assalto` e no desvio da
  `!igreja`) cobra uma
  multa de `fineMinPercent` a `fineMaxPercent` por cento do saldo e corta `arrestEaseCut` por cento
  da espera do `!fugir` que faltava (`lockUp`, via `easeEscape`), e as moedas somem, menos no
  `!prender`, em que o `arrest` (com `copKeepsFine`) credita a multa em quem mandou prender; a
  `!fianca` é outra coisa, é o preço de sair antes. Caminho novo que mande alguém pra cadeia precisa
  chamar o `lockUp` também, porque é ele que anota a prisão pra oferta de resgate da organização do
  crime (skill `economia-empresas`). Comando com `BarredInJail` no `economyCommands` é barrado pela guarda da
  economia (`economyGuard`) enquanto quem mandou tá preso ou em cativeiro (hoje `!apostar`,
  `!cassino`, `!blackjack`, `!crash`, `!corrida`, `!roubar`, `!furtar`, `!sequestrar`,
  `!venderrefem`, `!matar`, `!assalto`, `!quadrilha` e `!libertar`).
  A troca de profissão do `!profissao <nome>` também chama o `holdIfJailed` e o `holdIfHostage`
  (o comando não tem `BarredInJail` porque ver a lista e os detalhes continua liberado).
- O `!fugir` (`fugir.go`) tenta tirar quem tá preso da cadeia, com chance `escapeChance` mais o
  `escapeBonus` da profissão (negativo pra polícia) menos o `escape` do nível de fama
  (`celebrity()`, skill `economia-fama`), tudo no `escapeOdds`, que também vale pra quem ajuda no
  `!fugir @pessoa` e pra fuga do `!assalto`, uma tentativa a cada `escapeCooldown`
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
    ir preso na hora, por `baseJail` ajustado pelo `jailFactor`, com `lockUp` (passada pelo
    `flagrantOdds` de quem ajudou). No cativeiro, dar certo apaga a linha de `sequestros` e falhar não tem castigo.
- O `!libertar` (`libertar.go`) é o `!fugir` feito por outra pessoa: só funciona pra profissão
  com `releaseChance` (juiz, advogado, delegado), em quem tá preso ou em cativeiro (a cadeia vem
  primeiro), uma tentativa a cada `releaseCooldown` (coluna `libertacao_em`). Dar certo zera o
  `preso_ate` ou apaga a linha de `sequestros`, sem marcar crime em ninguém. Falhar tem
  `releaseFineChance` de cobrar uma multa entre `releaseFineMin` e `releaseFineMax` por cento do
  saldo de quem tentou, e as moedas somem (não vão pra ninguém). O presidente paga a multa com
  `presidentFineDiscount` de desconto (`releaseFineShare` da profissão).
- O `!denunciar` (`denunciar.go`) é o `!prender` de quem não é da polícia: usa o mesmo
  `walletStore.arrest` (mesma checagem de `crime_em`, mesmo resgate de refém), mas com chance
  `reportChance` mais o `reportBonus` de quem denuncia mais o `snitch` do nível de fama de quem é
  denunciado (`reportOddsOn`; o `reportOdds` sozinho é só pro texto da profissão), cadeia `baseJail` e espera
  `reportCooldown` (coluna `denuncia_em`). O `!prender` passa chance 1 pro `arrest`. Profissão
  com `noSnitch` (bandido) não denuncia: quem tenta é castigado na hora pelo `punishSnitch`, que
  sorteia um `snitchFate` com a mesma chance: cadeia (`jailSnitch`, `snitchSentence` com `lockUp`),
  morte (`killSnitch`, pelo `die`) ou cativeiro (`captureSnitch`, linha em `sequestros` por
  `hostageHold` com quem foi denunciado de `sequestrador`, então o `!prender` nele solta o X-9).
  Se o X-9 já é refém ou quem foi denunciado já tem um refém, o cativeiro vira cadeia. Os três
  soltam o refém que o X-9 tiver.
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
  vende e o `ransom` do nível de fama do refém (as duas porcentagens somam no `ransomOf`, que
  recebe a profissão do refém), e troca o `sequestrador` por `NULL` e corta o `ate` pra no máximo `soldHostageHold` a
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
  `BarredInJail`. Na mesma transação, depois do `claimBounty`, o `ripImplant` tem uma chance
  baixa de arrancar um implante da vítima e pôr na mochila de quem matou, desgastado (veja os
  implantes na skill `economia-dinheiro`); o
  Braço-canhão, implante clandestino, soma `killChance` e libera o `!matar` como uma arma, e a
  droga com `kill` no `high` (hoje o Pó de anjo) faz o mesmo enquanto tá batendo. Se der
  errado, tem `killCaughtChance` de ir preso na hora (mais `copVictimCaught` se a vítima tem
  `jailTime`), por `baseJail` ajustado pelo `jailFactor`.
- A `!recompensa @pessoa <valor>` (`recompensa.go`) desconta o valor de quem bota na hora e guarda
  na tabela `recompensas` (`chat`, `alvo`, `autor`, `valor`, `em`), uma linha por autor e alvo
  (botar de novo soma). Quando o `!matar` dá certo, o `kill` chama o `claimBounty` na mesma
  transação: soma tudo que tem na cabeça da vítima, apaga as linhas e credita no matador, então
  vale também pra cobrança do `!agiota` e pro deus. A `!recompensa retirar @pessoa` devolve só a
  parte de quem mandou. Nenhum outro jeito de morrer (overdose, saúde zerada) paga a recompensa.
  - O procurado do dia (`procurado.go`) é uma recompensa do próprio bot: o `runWanted`, a partir
    das `wantedHour` (BRT), pega cada economia que ainda não tem linha em `procurados` (`chat`,
    `dia`), marca o dia, apaga a recompensa que o bot tinha botado antes (linha com `autor` =
    `wantedAuthor`, que nenhum `!recompensa retirar` alcança) e bota de `wantedMin` a `wantedMax`
    moedas novas em alguém sorteado entre quem tá vivo e usou `!daily`, `!trabalhar` ou `!roubar`
    nas últimas `wantedActive` (`wantedSuspects`). O sorteio pesa pelo saldo: cada um entra com peso
    `saldo + wantedWeightFloor`, então quem tem mais dinheiro é o alvo mais provável, mas quem tá
    liso ainda pode sair. O aviso sai pelo `announce`, então só nos grupos com o
    submodo `anuncios`, e o `!recompensa` sem marcação mostra o procurado do dia embaixo da lista. Quem mata leva pelo `claimBounty` de sempre, junto do
    resto da cabeça.
  - O Quepe da Volante (relíquia passiva, `posse` na profissão) caça o procurado do dia. O
    `!matar` lê em paralelo com o `rivalsWithDebt` o prêmio do bot na cabeça da vítima
    (`wantedPrize`, pela chave da conta), e com `posse` e prêmio é caçada (`manhunt`): libera o
    `!matar` pra qualquer profissão, com pelo menos `posseKillChance` menos o `guard` da vítima,
    e é legal como a cobrança do `!agiota` (`lawful`, sem `crime_em` e sem flagrante, texto do
    `lawfulKillText`). Se der certo, o `posseReward` credita de novo o prêmio do bot em moeda
    nova, antes do `claimBounty` apagar as linhas. O `hunts` soma `posseLampiaoBonus` no
    `killOdds` (e na caçada) quando a vítima tem o Chapéu de Lampião (`lampiaoItem`), lido pelo
    `luxuries` da profissão dela.
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
  não zerar quem tem pouco), e as moedas somem, a não ser que o `funeral` tenha `heir` (a Foice).
  Antes disso, quem morre com prêmio fresco de aposta (alvo fácil, veja abaixo) perde ainda de
  `jackpotDeathMinPercent` a `jackpotDeathMaxPercent` por cento do prêmio (`spillJackpot`), somado
  no `death.lost`, então o `funeralText` e o `heir` já contam essa parte. Hoje chamam ele o `kill` do `!matar`, a overdose do
  `takeDrug`, o `hurt` da saúde zerada e o `!suicidio` (`suicidio.go`, sem espera, chance nem
  vantagem: é só zoeira, e a resposta fecha com o CVV 188, como o `!matar` em si mesmo, que
  continua recusado). Caminho novo de morte usa o `die` e mostra o `funeralText`.
- A saúde (`saude.go`) fica nas colunas `saude`/`saude_em` da `carteiras`: vale o número guardado
  mais 1 ponto a cada `healthRegenEvery` desde o `saude_em`, até `MaxHealth`, calculado na leitura
  (`readHealth`), sem nada rodando em segundo plano. Um enfermeiro do `!empregados` encurta esse
  intervalo (`regenEvery`, com o `care` do melhor enfermeiro), e como é lido na hora, contratar ou
  demitir muda também o tempo que já passou desde o `saude_em`. Quem muda a saúde grava os dois juntos
  (`setHealth`). O `!trabalhar` das profissões com `hazard` (as perigosas de obra, transporte,
  cozinha, campo, polícia, crime...) e as quatro extrações (o `hazard` de cada `activity`)
  sorteiam um ferimento (`hazard.roll`, dano de `harm/2` a `harm`), e o `hurt` aplica na mesma
  transação, só quando o comando saiu de fato (com espera pendente ninguém se machuca). A chance
  cai pela porcentagem de `safety`, até `maxSafety`: no trampo, o `safety` da profissão (o Kit EPI,
  pelo `flex`); nas extrações, o `safety` do `gear` que cabe nela (capacete, luva de borracha,
  boia salva-vidas e roupa de apicultor), somado no `equip`. Saúde zerada mata pelo `die`. Item
  com `heal` (tipo `Saúde`) cura na hora no `!usar` (`walletStore.heal`, que recusa com a saúde
  cheia sem gastar). O Curativo e o Kit de primeiros socorros acabam no primeiro uso; os que têm
  `doses` (Garrafada da benzedeira, Maleta de médico, Ampola de adrenalina e Água benta de
  Aparecida) gastam uma dose por uso no `spendDose`, contada na coluna `usos` do `inventarios`
  pelo `wearTool` (a mesma do desgaste das ferramentas, então o `moveItems` leva as doses gastas
  junto), e só tiram a unidade da mochila na última, zerando o `usos` pra próxima. O `!usar`
  mostra quantas doses sobram na unidade aberta (`dosesText`). O `!hospital` de quem tá
  vivo e machucado (`treat`) enche a saúde cobrando o `hospitalPrice` vezes a fração que falta
  (`carePrice`). O `!saude` mostra a saúde e a chance de se machucar em cada lugar, já com a
  proteção. Profissão perigosa nova é só ganhar um `hazard`; os `wounds` entram logo depois do
  `@pessoa` (tipo "caiu do andaime"), então começam com verbo no passado.
- Os crimes também ferem (`ferimentos.go`): cada um tem um `crimeRisk` (`robRisk`, `theftRisk`,
  `kidnapRisk`, `killRisk`, `heistRisk`, `gangRisk`, listados em `crimeRisks` pro `!saude`) com o
  `self` de quem comete e o `victim` de quem sofre. O `self` vale em toda tentativa que saiu de fato
  (alarme e cadeado contam como dar errado), com a chance vezes `botchedRiskFactor` quando dá
  errado, passa pelo `hurt` e mata se zerar a saúde, como no trampo: quem morre assim não é preso
  no flagrante, não aparece no `fate` do `!assalto` e não dispara evento. O `victim` só vale quando
  o crime machuca alguém de fato: no `!roubar`, no `!sequestrar` e na `!quadrilha` quando dá certo,
  e no `!matar` quando falha sem o colete segurar (tiro de raspão); o `!furtar` e o `!assalto` não
  têm vítima. A vítima passa pelo `bruise`, que nunca deixa a saúde abaixo de 1, pra roubo não
  virar um `!matar` sem recompensa nem cadeia. O `clash` aplica os dois na transação do crime, e o
  `by`/`on` zeram o lado de quem é deus. Crime novo ganha um `crimeRisk`, entra no `crimeRisks` e
  chama o `clash` (ou o `hurt` por pessoa, nos crimes em equipe) depois de decidir o resultado.
- O `!tacar @pessoa [pedra|tomate]` (`tacar.go`, aliases `!tacarpedra` e `!tacartomate`) não é
  crime: não marca `crime_em` nem `ficha_em` e não tem flagrante. Cada `projectile` (`projectiles`,
  hoje a Pedra e o Tomate) é um item do `catalog` que a pessoa precisa ter na mochila, com o dano
  (`harm`) e os textos dele; o `aimedProjectile` escolhe pelo id que aparece no texto (sem nenhum,
  pedra). O `walletStore.throw` carimba a espera `throwCooldown` (coluna `arremesso_em`, passada
  pelo `pace`) e gasta uma unidade do item na mesma transação; sem o item, o `errNoItems` desfaz a
  espera. Acerta com `throwHitChance` menos o `guard` da vítima, pelo `sober` de quem taca, e
  depois, acertando ou não, a vítima revida com `throwRetortChance` com o mesmo dano. Os dois lados
  passam pelo `bruise`, então ninguém morre de pedrada, e o deus não sofre dano nem revide.
  Projétil novo é só um item no `catalog` e uma entrada no `projectiles`.
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
- O alvo do `!roubar`, `!furtar`, `!sequestrar`, `!matar`, `!quadrilha`, `!prender` e `!denunciar`
  sai do `targetOf` (`alvo.go`), nesta ordem: a marcação, o número do contato (vcard, pelo `core.QuotedContact`) da mensagem
  respondida, o autor da mensagem respondida (menos o próprio bot), o primeiro `@número` digitado e,
  por último, o nome (`personNamed`): o começo do texto depois do comando comparado com os nomes do
  `!nome` (inclusive os automáticos) e os `!apelido` da conversa, sem acento e sem maiúscula, ficando
  com o nome mais comprido que bate. O `target.rest` é o texto sem o nome, que o `!furtar` usa pra
  achar o item. Nome que ninguém tem ou que mais de uma pessoa tem volta um texto de recusa, menos na
  `!quadrilha` com uma quadrilha aberta, em que vira entrar nela. Nada disso olha quem tá no grupo,
  então dá pra mirar quem saiu, que o WhatsApp não deixa marcar. Comando novo que mire uma pessoa
  usa o `targetOf` e mostra o `targetHint` quando não vem ninguém.
- A `!quadrilha @pessoa` (`quadrilha.go`) é o `!roubar` em grupo: junta até `gangSize` pessoas em
  `gangs` (uma por conversa, efêmera e salva com `KeepGame`) contra uma vítima só, marcada por quem
  abre; quem entra depois manda `!quadrilha` (marcar outra pessoa é recusado). Com `gangSize` sai
  na hora; se a `gangWindow` acabar com pelo menos `gangMin`, sai assim mesmo, e com menos é
  desfeita. A espera é a do `!roubar` (`robCooldown()` de cada um), checada ao entrar e carimbada
  pra todo mundo na hora do roubo, junto do `crime_em`. A chance é `robSuccessChance` mais o
  `robBonus` de cada um mais `gangHelp` por pessoa a mais, menos o `guard` da vítima e o toque de
  recolher. O alarme da vítima (`soundAlarm`, em `carteiras.go`) dispara menos que no `!roubar`:
  lá a chance é `alarmChance`, e aqui o `gangAlarmChance` tira `alarmGangCut` por pessoa a mais. Se der certo, a quadrilha leva de
  `gangLootMin` a `gangLootMax` por cento do saldo da vítima, passado pelas faixas do `exposed` (veja o valor de roubo abaixo), dividido igual, e a parte de cada um cresce o `robHaul` da profissão dele (hoje só o Chapéu de Lampião, o mesmo que aumenta o que leva no `!roubar`); se der errado, cada
  um paga de 10% a 20% do próprio saldo pra vítima, e quem tinha menos de `brokeThief` (o `broke`
  do `indemnify`) tem `brokeCaughtChance` de ir preso na hora pelo `jailBroke`, com as mesmas regras
  do ladrão liso do `!roubar` (veja abaixo).
- Roubo de saldo tem duas travas (`escudo.go`, `ficha.go`):
  - O escudo: quando um `!roubar` ou uma `!quadrilha` dá certo, o `raiseShield` grava
    `escudo_ate` (agora mais `robShield`, ou menos se ela tá marcada como alvo fácil, veja abaixo) na vítima, e enquanto ele não passa o `shieldOn` barra
    os dois antes de carimbar espera ou marcar crime de alguém. A `!quadrilha` checa o escudo
    também ao abrir (`shieldLeft`). O `!furtar` não olha o escudo. O `!saldo` e o `!tu` mostram o
    tempo que falta (`shieldText`).
  - O ladrão liso: no `!roubar` que falha (sem alarme e sem morrer no `clash`), quem tinha menos de
    `brokeThief` não tem como pagar a indenização e pode ir preso na hora, com chance
    `flagrantOdds(brokeCaughtChance)` (o Gorro escapa) e cadeia de `baseJail` ajustada pelo
    `jailFactor`, pelo `jailRobber`, que chama o `lockUp` e zera o `crime_em`. O `flagrant` leva a
    chance e a pena pro `walletStore.rob`. Quem escapa paga o que der da indenização, como qualquer
    ladrão que falha.
  - A ficha limpa: o `ficha_em` da `carteiras` guarda a hora do último crime e, diferente do
    `crime_em`, a prisão e a anistia não zeram ele. Todo lugar que marca o `crime_em` com a hora
    (`markCrime`, `stampGang`, o `!assalto` e as fugas do `fugir.go`) grava o `ficha_em` junto, e
    caminho novo que marque crime precisa fazer o mesmo. O `ProfessionOf` lê ele no `standing` e
    aplica o `cleared` logo depois do `escorted`: cada `cleanEvery` sem crime soma `cleanStep` no
    `guard` (guardado em `clean`), até `cleanMax`. Quem nunca cometeu crime (`ficha_em` 0) tem a
    ficha cheia. O `cleanText` mostra a ficha no `!saldo` e no `!tu`.
- O valor de roubo (`exposed` em `economia.go`) é o quanto do saldo um roubo consegue levar. A
  porcentagem sorteada (`robLootMin` a `robLootMax` no `!roubar`, `gangLootMin` a `gangLootMax` na
  `!quadrilha`) vale cheia só nas primeiras 1.000 moedas e cai nas faixas de cima, como imposto de
  renda (3/4 até 10 mil, metade até 100 mil, 1/4 até 1 milhão e 1/10 dali pra frente), então
  rico perde mais moedas que pobre, mas uma fatia bem menor do saldo. O `robHaul` de quem rouba
  multiplica depois das faixas. O `exposureText` mostra o máximo de um `!roubar` no `!saldo` e no
  `!tu`. Roubo novo que tire porcentagem do saldo de alguém passa pelo `exposed`.
- Quem acabou de ganhar muito em aposta vira alvo fácil (`alvofacil.go`, colunas `premio_aposta` e
  `premio_aposta_em` da `carteiras`), mas só se tiver no lucro. O `NoteBet` chama o `noteWinnings`,
  que numa transação só soma o lucro líquido da aposta na tabela `apostas_lucro` (`tallyWinnings`,
  veja a marca do `lucky` abaixo) e guarda o lucro de toda aposta de pelo menos `jackpotMin`
  (`stashJackpot`), somando com o prêmio anterior se ele ainda tá dentro da `jackpotWindow`. No `!roubar` (`rob`) e na `!quadrilha` (`gangRob`), o `markOf` lê a marca (`easyMark`) dentro da transação, com o
  prêmio pelo `freshJackpot`, que fica com o menor entre o prêmio, o saldo e o lucro de aposta dos
  últimos `luckyDays` dias, e dá zero se o prêmio venceu ou se esse lucro tá abaixo de `jackpotMin`.
  Então quem tá no prejuízo não vira alvo fácil mesmo ganhando uma aposta grande, e quem perde o
  lucro dentro da janela perde a marca: a espera do roubo de quem
  ataca cai `jackpotWaitCut` (`jackpotPace`, lido antes do `stampOrAbuse` no `rob` e no
  `robWait` da entrada da `!quadrilha`, que acha a vítima pelo `gangVictim`), a chance ganha
  `jackpotRobBonus` (`easyMark.odds`), o alarme dela perde `jackpotAlarmCut` de chance de disparar
  (`jackpotAlarm`, por cima do `alarmChance` ou do `gangAlarmChance`) e o `looted` passa o saldo
  todo pelas faixas do `exposed`, como num roubo normal, e só por cima do total soma
  `jackpotLootBonus` (na `!quadrilha`, antes de dividir), então o prêmio não sai com a porcentagem
  cheia. O roubo que dá certo não apaga o prêmio:
  a marca fica até a `jackpotWindow` acabar, e o escudo que a vítima ganha é o `jackpotShield`,
  `robShield` (sempre bem menor que a `jackpotWindow`, senão a marca venceria antes do escudo
  acabar) com um corte que cresce com o prêmio (`jackpotShieldCut`): `jackpotShieldMinCut` no
  `jackpotMin`, subindo em escala logarítmica até `jackpotShieldMaxCut` no `jackpotShieldTop` ou
  mais, arredondado pro segundo. O `easyMark.text` avisa na resposta. Quando a marca começa (o
  `noteWinnings` não achou prêmio fresco antes de gravar e acha depois, inclusive quando a aposta
  tira a pessoa do prejuízo), o `NoteBet` manda no grupo o `markedText` com o prêmio que vale,
  marcando a pessoa; somar mais prêmio na mesma janela não avisa de novo. Por isso
  o `NoteBet` roda depois da resposta da aposta (no `!roleta`, depois do resultado do suspense). Morrer com o
  prêmio fresco, de qualquer jeito, também custa uma parte dele (`spillJackpot` no `bury`, acima).
  O prêmio fresco também ajuda: o `!prender` nessa pessoa falha `jackpotArrestSlip` das vezes
  (`slipsArrest`, o advogado pago com o prêmio), e a marca mais fraca do `lucky` não conta.
  Depois que o prêmio vence, sobra uma marca mais fraca de `luckyDays` dias: o `tallyWinnings` soma
  o lucro líquido de toda aposta (perda desconta) por dia (BRT) na tabela `apostas_lucro` e apaga
  os dias velhos da pessoa. Sem prêmio fresco, o `markOf` soma os
  últimos `luckyDays` dias (`luckySince`) e, com `luckyMin` ou mais, liga o `lucky` do `easyMark`,
  que só soma `luckyRobBonus` na chance (espera, alarme, escudo e valor levado ficam normais) e
  troca o aviso da resposta. Com prêmio fresco vale só a marca do alvo fácil, sem somar as duas. Não confundir com o evento
  `bolada` (`bolada.go`), que olha o saldo em memória e não mexe em roubo. Enquanto a marca durar
  e o ganho dela (`easyMark.haul`) for do tamanho de uma bolada, a pessoa também pode cair nos
  eventos pesados da bolada (sequestro relâmpago, quadrilha...) pelo `stirStalker`, com chance
  menor que a da bolada (skill `economia-eventos`). Quem tem contrato com uma empresa de segurança
  pode ter a marca anulada em cada roubo: o `escortMark`, logo depois do `markOf` no `rob` e no
  `gangRob`, sorteia e, se der, zera o `easyMark` antes de tudo que depende dele (skill
  `economia-empresas`).
  A marca com prêmio fresco tem nível (`nivelalvo.go`, colunas `alvo_nivel` e `alvo_apostado` da
  `carteiras`). Ela começa no 1 (o `noteWinnings` chama o `resetMarkLevel` quando a marca começa) e
  o `NoteBet`, antes de tudo, passa o valor apostado pelo `noteMarkLevel`: com a marca ligada, o
  `markLevel.after` soma no `alvo_apostado` e, a cada vez que a soma chega no prêmio fresco, sobe um
  nível (vários de uma vez se a aposta for grande), até `markLevelMax`, e avisa no grupo. O `markOf`
  lê o nível no `easyMark.level` e o `steps()` (nível menos 1) mexe em três lugares: o
  `easyMark.odds` soma `levelRobStep` por passo no `jackpotRobBonus`, o `contract.odds` da empresa
  perde `levelEscortCut` por passo, e o `stirStalker` (skill `economia-eventos`) multiplica a chance
  e divide a espera pelo nível. A marca fraca do `lucky` não tem nível. O `!bicho` e o `!bolao` não
  passam pelo `NoteBet` e não sobem nível.
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
- O flagrante na hora (`!sequestrar`, `!matar`, `!fugir @pessoa`, o ladrão liso do `!roubar` e da
  `!quadrilha`, o `jailCrew` do `!assalto`, o desvio da `!igreja` e o `nabbed` dos eventos) passa
  pelo `flagrantOdds(chance)` (`prender.go`):
  zero pro `elusive` do Gorro do Saci, a chance vezes `1 - immunity()` pro presidente (skill
  `economia-presidencia`) e a chance cheia pro resto. Crime que dá errado nunca prende de certeza:
  a chance base do flagrante de cada um (`brokeCaughtChance`, `kidnapCaughtChance`,
  `killCaughtChance`, `rescueCaught`) fica entre o `reportChance` do `!denunciar` e 1, o que o
  `TestBotchedCrimeCaughtOdds` confere; crime novo com flagrante segue a mesma faixa. No `jailCrew`, quem escapou assim fica com
  `shielded`, que escolhe o texto do `fate`. O Gorro só escapa do flagrante, não do `!prender`, do
  `!denunciar` nem da batida do `!mercadonegro`. Não tem jeito de comprar a polícia nem o juiz: o
  `!processar` sempre sai pela enquete do júri.
