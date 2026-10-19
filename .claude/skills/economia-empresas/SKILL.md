---
name: economia-empresas
description: "Empresas de NPC da economia (empresa.go, escolta.go, reforco.go e recuperacao.go, !empresa lista/contratar/cancelar, catálogo firms em código, tabela contratos com ramo, contractOf e contractText, diária fee com piso e taxa de emergência, escortMark no !roubar e na !quadrilha, cushion dos eventos bolada/alvofacil, oferta de reforço offerBoost depois do prêmio, oferta de recuperação offerRecovery depois do roubo, boostOffers/recoveryOffers respondidos pelo !aceitar/!recusar com answerFirmOffer). Carregue antes de mexer numa empresa, num contrato ou numa oferta delas."
user-invocable: false
---

# Economia: empresas

- As empresas são de NPC e vivem em código (`firms` em `empresa.go`); só o `id` vai pra coluna
  `empresa` da tabela `contratos`. Nunca renomeie um `id`: o contrato some (o `contractOf` devolve
  `found` falso). Hoje só existe o ramo de segurança (`securitySector`), mas a tabela já guarda o
  `ramo`, com chave (`chat`, `pessoa`, `ramo`), então ramo novo é só outra constante e outra lista,
  sem migração. É um contrato por pessoa por ramo.
- Cada empresa de segurança tem `annul` (chance de anular a marca de alvo fácil num roubo), `gangs`
  (se cobre a `!quadrilha` além do `!roubar`), `cover` (porcentagem da perda coberta nos eventos) e
  a diária: `rate` por cento do saldo com piso `floor` (`fee`), e as moedas somem (é ralo). O
  `perks()` monta o texto do `!empresa` e da contratação a partir desses campos, então mudar um
  número já muda o texto.
- O `!empresa contratar <nome>` (`walletStore.sign`) cobra a diária e grava `ate` em
  `contractSpan`. Contratar a mesma empresa com o contrato ativo soma `contractSpan` no fim dele;
  trocar de empresa começa um contrato novo a partir de agora (o tempo que sobrava se perde). Quem
  tá com prêmio fresco de aposta (`freshJackpot`, skill `economia-crime`) paga ainda
  `emergencyCut` por cento dele de taxa de emergência (`hiring.urgency`). O `!empresa cancelar`
  apaga a linha sem devolver nada. Contrato não é espera: fica fora do `resetCooldowns`.
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
- **Ofertas** ficam em memória (`boostOffers` e `recoveryOffers` no `bot`, chave `OfferKey`) e
  vencem em `offerWindow` (o mesmo prazo da venda do `!vender`); o `takeOffer` descarta a vencida.
  O `!aceitar` e o `!recusar` (`vender.go`) chamam o `answerFirmOffer` depois da venda e do
  empréstimo. Quem não tem contrato recebe a oferta da `offerFirmID` (Escolta armada).
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
- O `!saldo` e o `!tu` mostram o contrato ativo (`contractText`), e o `!empresa` sem nada lista as
  empresas com a diária calculada pro saldo de quem pediu.
- Teste: `go test ./internal/economy/ -run 'TestSignFirm|TestCancelContract|TestEscort|TestEventCushion|TestBoostContract|TestRecover|TestTakeOffer|TestMarkLevel|TestStalk'`.
