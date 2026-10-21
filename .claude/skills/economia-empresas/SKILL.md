---
name: economia-empresas
description: "Empresas de NPC da economia (empresa.go, escolta.go, reforco.go, recuperacao.go e organizacao.go, !empresa lista/crime/contratar/cancelar, catálogos firms e syndicates em código, ramos seguranca e crime, organizações do crime com bônus no !roubar/!assalto/!quadrilha e oferta de resgate da cadeia breakoutOffers depois do lockUp, tabela contratos com ramo, contractOf e contractText, diária fee com piso e taxa de emergência, escortMark no !roubar e na !quadrilha, cushion dos eventos bolada/alvofacil, oferta de reforço offerBoost depois do prêmio, oferta de recuperação offerRecovery depois do roubo, boostOffers/recoveryOffers respondidos pelo !aceitar/!recusar com answerFirmOffer). Carregue antes de mexer numa empresa, num contrato ou numa oferta delas."
user-invocable: false
---

# Economia: empresas

- As empresas são de NPC e vivem em código; só o `id` vai pra coluna `empresa` da tabela
  `contratos`. Nunca renomeie um `id`: o contrato some (o `contractIn` devolve `found` falso). Tem
  dois ramos, cada um com a constante e a lista dele: segurança (`securitySector`, `firms` em
  `empresa.go`) e crime (`crimeSector`, `syndicates` em `organizacao.go`). Cada `firm` guarda o
  `sector` dela, e o `allFirms` junta as duas listas pro `firmByID` e o `findFirm`, então os `id`s
  e nomes não podem se repetir entre os ramos. A chave da tabela é (`chat`, `pessoa`, `ramo`): um
  contrato por pessoa por ramo, então dá pra ter um de segurança e um do crime ao mesmo tempo.
  Ramo novo é outra constante, outra lista no `allFirms` e outra entrada no `sectors()`, sem
  migração. O `contractIn` lê o contrato de um ramo (o `contractOf` é ele com a segurança, o
  `syndicateOf` devolve só a organização do crime) e o `walletStore.contracts` lê todos, com a
  segurança primeiro.
- Cada empresa de segurança tem `annul` (chance de anular a marca de alvo fácil num roubo), `gangs`
  (se cobre a `!quadrilha` além do `!roubar`), `cover` (porcentagem da perda coberta nos eventos) e
  a diária: `rate` por cento do saldo com piso `floor` (`fee`), e as moedas somem (é ralo). O
  `perks()` monta o texto do `!empresa` e da contratação a partir desses campos (pra organização
  do crime, pelo `backing()`), então mudar um número já muda o texto.
- O `!empresa contratar <nome>` (`walletStore.sign`) cobra a diária e grava `ate` em
  `contractSpan`, no ramo da empresa escolhida. Contratar a mesma empresa com o contrato ativo soma
  `contractSpan` no fim dele; trocar por outra do mesmo ramo começa um contrato novo a partir de
  agora (o tempo que sobrava se perde). Quem contrata segurança com prêmio fresco de aposta
  (`freshJackpot`, skill `economia-crime`) paga ainda `emergencyCut` por cento dele de taxa de
  emergência (`firm.emergency`, só no ramo de segurança). O `!empresa cancelar [nome]` apaga a
  linha do ramo sem devolver nada: sem nome, cancela o único contrato que a pessoa tiver e, com
  dois, pede o nome (`cancelledSector`). Contrato não é espera: fica fora do `resetCooldowns`.
- O `!empresa` sem nada lista os dois ramos (`sectors()`, cada um com título, lista e rodapé), e o
  `!empresa seguranca` e o `!empresa crime` listam um só.
- **Organizações do crime** (`organizacao.go`): cada uma tem `edge` (chance somada no crime),
  `gangs` (se o `edge` vale também no `!assalto` e na `!quadrilha`), o resgate (`breakout` é o
  preço fixo, que não depende do saldo, e `breakoutOdds` a chance de dar certo) e o `stash`, a
  porcentagem do valor da muamba apreendida que ela cobra pra devolver.
  - Bônus: o `!roubar` lê o `syndicateOf` do ladrão em paralelo com as profissões e soma o `edge`
    na chance antes do `sober`. Na `!quadrilha` (lido em paralelo no `runGang`) e no `!assalto`
    (lido dentro da transação do `heist`), o `crewSyndicate` fica com a melhor organização com
    `gangs` da equipe, sem somar as de cada um. Quando o crime dá certo, o `coverText` abre a
    resposta dizendo quem deu cobertura.
  - Resgate: o `lockUp` (toda ida pra cadeia, skill `economia-crime`) anota a pessoa no
    `ledgerTx.jailed`, e o `inTx` passa pro `walletStore.jailings` (`jailLog`, por conversa) só
    depois do commit. O `offerBreakouts`, no `AddAfterMessage` depois do `announcePayouts`, tira as
    prisões daquela conversa (`jailing`) e, pra quem tem contrato no ramo do crime, guarda um
    `breakoutOffer` em `breakoutOffers` (chave `OfferKey`, vence em `offerWindow`) com duas partes:
    a fuga (`escape`, só se ainda tá preso) e a muamba (`contraband`, só se a polícia apreendeu
    alguma coisa). Quem não tem contrato não recebe oferta. O `!aceitar` sozinho fecha as duas, e o
    `!aceitar fuga` ou `!aceitar muamba` fecha uma só (`picked`); a outra parte volta pro
    `breakoutOffers` com o mesmo prazo (`without`), e o `!recusar` dispensa tudo.
  - Fuga: o `walletStore.breakout` cobra o `breakout` com o
    `spend` e, em `breakoutOdds` das vezes, zera o `preso_ate` e marca `crime_em` e `ficha_em`
    (é fuga, então a polícia pode prender de novo); dando errado, as moedas somem e a pessoa fica
    presa. Se ela já saiu, o `errNotHeld` desfaz a cobrança.
  - Muamba: o `seizeContraband` (`mercadonegro.go`, chamado pelo `arrest` do `!prender`/`!denunciar`
    e pela batida do `!mercadonegro`, sempre depois do `lockUp`) devolve as pilhas com a qualidade de
    cada unidade e anota elas na prisão da mesma transação (`ledgerTx.confiscated`). O
    `walletStore.reclaim` cobra o `stashPrice` (o `stash` por cento do `worth` das pilhas) com o
    `spend` e põe tudo de volta na mochila com o `addGrades`, sem chance de dar errado. Os usos
    gastos (doses, desgaste) não voltam, e o limite de mochila (`trimCarry`) vale como sempre.
- **Anulação da marca** (`escortMark` em `escolta.go`): o `walletStore.rob` e o `gangRob` chamam
  logo depois do `markOf`, então vale antes do `jackpotPace`, do `stampOrAbuse`, do alarme, do
  `looted` e do `jackpotShield`. Só age com a marca ligada (prêmio fresco ou `lucky`). Sorteia uma
  vez por tentativa de roubo, com o `contract.odds`: o `annul` cheio com prêmio fresco, metade no
  `lucky`, mais `boostOdds` com o reforço ativo, menos `levelEscortCut` por nível da marca acima do
  1 (skill `economia-crime`), até `escortOddsMax`. A `!quadrilha` só passa pela
  empresa com `gangs`. Quando anula, a marca vira `easyMark{}` e o roubo segue como contra qualquer
  um; a empresa vai pro `robbery.escort`/`gangRobbery.escort` e o `escortText` abre a resposta.
- **Cobertura nos eventos** (`eventRun.cushion`): o `loseEffect` e o `skimEffect`, só nos lugares
  `windfallPlace` e `stalkPlace` (skill `economia-eventos`), perguntam quanto o contrato cobre
  (`cover` por cento do que ia sumir). No `lose` a parte coberta volta pro saldo depois do
  `burnBalance`; no `skim` ela já sai descontada. A linha do efeito mostra quanto a empresa segurou.
  Efeito novo de perda em moeda desses lugares chama o `cushion` também.
- **Ofertas** ficam em memória (`boostOffers`, `recoveryOffers` e `breakoutOffers` no `bot`, chave `OfferKey`) e
  vencem em `offerWindow` (o mesmo prazo da venda do `!vender`); o `takeOffer` descarta a vencida.
  O `!aceitar` e o `!recusar` (`vender.go`) chamam o `answerFirmOffer` depois da venda e do
  empréstimo. Quem não tem contrato de segurança recebe o reforço e a recuperação da `offerFirmID` (Escolta armada).
  - Reforço (`reforco.go`): o `noteWinnings` de `quedas.go`, logo depois do aviso 🎯 do
    `markedText`, chama o `offerBoost`, que só oferece com prêmio de pelo menos `boostPrizeMin`.
    Custa `boostCut` por cento do prêmio, e o `walletStore.boost` grava o `reforco_ate` até a marca
    acabar (`jackpotWindow`): o cliente continua com a empresa e o contrato dele (o `ate` só cresce
    se ia vencer antes), e quem não era cliente ganha um contrato da Escolta armada que vence junto
    com a marca.
  - Recuperação (`recuperacao.go`): o `!roubar` que deu certo chama o `offerRecovery`, que oferece
    em `recoveryOfferChance` das vezes pra vítima. O preço é `recoveryCut` por cento do que foi
    roubado, mais caro com a marca (fraca e com prêmio fresco), lido do `robbery.mark` (que já vem
    zerado se a empresa anulou). O `walletStore.recover` cobra adiantado com o `spend` e, em
    `recoveryOdds` das vezes, tira do saldo do ladrão o que ele ainda tiver, até o valor roubado.
    Nenhuma moeda é criada: a cobrança some e a recuperação só move moeda do ladrão pra vítima.
- **Aviso de nível**: o `PlaceBet`, depois do `placeStake`, chama o `markWarning`. Pra quem tem
  contrato e tá com a marca, a aposta que subiria o nível (`markLevel.after`) não sai de primeira:
  a empresa avisa e a pessoa confirma mandando o mesmo comando em até `allInWindow`
  (`markWarnings`, um `allInBoard` com chave `eventKey` mais a sintaxe).
  Quem não tem contrato sobe de nível sem aviso. O `!bolao` usa o `placeStake` direto e fica de
  fora, porque não passa pelo `NoteBet`, onde o nível sobe.
- O `!saldo` e o `!tu` mostram os contratos ativos dos dois ramos (`contractText` com o
  `walletStore.contracts`), e o `!empresa` lista as empresas com a diária calculada pro saldo de
  quem pediu.
- Teste: `go test ./internal/economy/ -run 'TestSignFirm|TestCancelContract|TestEscort|TestEventCushion|TestBoostContract|TestRecover|TestTakeOffer|TestMarkLevel|TestStalk|TestSignSyndicate|TestCrewSyndicate|TestLockUpNotesJailing|TestBreakout|TestSeizureRidesJailing|TestReclaimContraband'`.
