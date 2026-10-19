---
name: economia-eventos
description: "Eventos com escolha da economia (motor em evento.go, efeitos em eventoefeitos.go, diário !eventos em eventos.go, livros de conteúdo eventos*.go, scenario/choice/ending/effect, stirEvent nos comandos, enquete que só conta o voto do dono, KeepGame do tipo evento, tabelas eventos_vistos e eventos_marcados, anti-repetição, etapas next e plant). Carregue antes de mexer num evento, criar um novo ou ligar o gatilho num comando."
user-invocable: false
---

# Economia: eventos com escolha

Depois que o `!trabalhar`, uma extração (`!pescar`, `!minerar`, `!vasculhar`, `!catar`) ou um crime
(`!roubar`, `!furtar`, `!sequestrar`, `!matar`, `!assalto`, `!quadrilha`, `!mercadonegro`) sai de fato,
ou depois de um `!postar` ou um `!publi` que saiu (sem espera pendente), o
comando chama o `b.stirEvent` (ou o `stirCrewEvent` nos crimes em grupo), que em `eventChance` das vezes abre
um evento: o bot manda o texto da situação com a @pessoa e uma enquete com as opções liberadas pra ela.

- **Gatilho.** O `stirEvent` vai logo depois da mensagem do resultado, só quando a atividade aconteceu (sem
  espera pendente, sem recusa, sem a pessoa morrer ou ser presa em flagrante ali mesmo). Ele recebe um
  `eventSpot` com o `place` (o comando), o `won` (a atividade deu bom: achou item, não deu ruim no trampo, o
  crime deu certo, o post não flopou nem foi cancelado, a publi não teve backlash) e a `victim` dos crimes com vítima. O `!assalto` sorteia o evento entre quem não foi preso
  (`uncaught`) e a `!quadrilha` entre os membros. Comando novo que solte evento segue o mesmo formato, e o
  `place` dele entra no `eventPlaces` do teste e, se tiver espera, no `placeCooldown`.
  O `eventLuck` sorteia `eventChance` vezes `oracleEventFactor` antes de ler qualquer coisa, e o
  `openEvent`, depois do `eventCaseOf`, só segue pra quem não tem os Búzios encantados em 1 de cada
  `oracleEventFactor` vezes (`catchesEvent`). Assim quem não tem fica com o `eventChance` de sempre, quem
  tem fica com o triplo, e a profissão só é lida quando o primeiro sorteio passa.
- **Prejuízo nas apostas** (`prejuizo.go`). As apostas (`!apostar`, `!cassino`, `!blackjack`, `!crash` e
  `!corrida`) não soltam evento no sorteio normal: cada aposta fechada chama o `tallyBet` (o `!crash` e a
  `!corrida` chamam o `tallyRound` no fim da rodada, com o `settled()` de cada aposta), que soma o prejuízo
  da pessoa naquela conversa no `tiltBoard`, em memória. A sequência começa com o saldo de antes da
  primeira aposta (`bankroll`), zera quando a pessoa volta pro lucro ou fica `tiltWindow` sem apostar, e
  dispara quando o prejuízo chega em `tiltShare`% do `bankroll` (no mínimo `tiltFloor`). Aí zera de novo
  e, com a mesma conta do `eventLuck` usando `tiltChance`, abre o evento com o `place` da aposta e
  `won` falso. O livro dessas apostas é o `gamblingEvents` (`eventosaposta.go`, "Fundo do poço"), que
  só aparece por esse caminho.
- **Apostar tudo** (`tudo.go`). Toda aposta que passa pelo `stakeAfterLevy` (`PlaceBet` e `PlaceStake`:
  `!apostar`, `!cassino`, `!blackjack`, `!crash`, `!corrida`, `!roleta`, `!bicho`, `!bolao` e o
  `!jokenpo @pessoa`) e pede o saldo inteiro (`stake` igual ao `budget`, seja `tudo`, `100%` ou o número
  exato) cai no `allIn`. Na primeira vez a aposta não sai: o bot avisa e guarda o pedido no `allInBoard`
  (em memória, chave `eventKey` mais o comando). Mandar o mesmo comando de novo em até `allInWindow`
  confirma. Na confirmação, quem aposta `allInFloor` ou mais tem `allInChance` (com a mesma conta do
  `eventLuck`) de cair num evento do `allInEvents` (`eventostudo.go`, place `allInPlace`) antes da aposta;
  se o evento abrir, a aposta é cancelada e a pessoa precisa confirmar de novo depois. O `!bicho` chama o
  `affordable` e só passa pelo `allIn` depois de conferir o limite do tipo de aposta. O `!lance` do leilão
  também usa o `affordable`, mas não passa pelo `allIn`.
- **Bolada** (`bolada.go`). O `stirWindfall`, registrado no `AddAfterCommand`, lê o saldo de quem mandou
  qualquer comando da economia num grupo e guarda no `windfallBoard`, em memória, a base de onde a pessoa
  partiu (`purse`): a base vale por `windfallWindow`, desce junto quando o saldo cai e, se o saldo subir
  `windfallShare`% da base (no mínimo `windfallFloor`), a base vira o saldo de agora e, com a mesma conta
  do `eventLuck` usando `windfallChance`, abre um evento com o `place` `windfallPlace` (`bolada`) e `won`
  verdadeiro. Por ser lido depois do comando, pega ganho de qualquer lugar (trampo, aposta, roubo,
  milagre, Pix). O livro é o `windfallEvents` (`eventosbolada.go`, "Dinheiro chama ladrão"): gente
  tentando roubar quem ficou rico, e o `bolada-casa` só cabe pra quem tem imóvel (`homeOwner`).
- **Cassino furioso** (`cassinofurioso.go`). O `stirCasinoWrath`, também no `AddAfterCommand`, roda
  depois de `!apostar`, `!cassino`, `!blackjack`, `!crash`, `!corrida` e `!roleta` e só faz alguma coisa
  com o decreto `cassino` positivo (o prêmio a mais; a taxa negativa não conta). A chance cresce com o
  nível do decreto (`level / casinoDecreeMaxRaise`, na conta do `eventLuck`): `casinoWrathChance` de abrir
  um evento pro presidente (`casinoWrathPlace`, no máximo um a cada `casinoWrathEvery` por conversa, no
  `wrathBoard`) e `casinoWatchChance` de abrir um pra quem apostou (`casinoWatchPlace`), menos quando quem
  aposta é o presidente. Os dois lugares ficam no livro `casinoWrathEvents` (`eventoscassino.go`), cada
  evento com o `places` dele. O efeito `repealCasino` zera o `cassino_pct` da `presidencias` se quem
  joga o evento for o presidente com o decreto ainda positivo, então só faz sentido nos eventos do
  presidente.
- **Escolha do evento.** O `pickScenario` dá prioridade a um evento marcado com `plant` que já venceu
  (`eventos_marcados`), senão sorteia pelo `weight` entre os que cabem (`fits`: `places`, `areas`, `jobs`,
  `when` e pelo menos 2 opções liberadas). Os lugares dos livros especiais (apostas, `tudo`, `bolada` e
  os do cassino furioso, juntos no `sealedPlaces`) são fechados: lá só sai evento que cita o lugar no
  `places` (`covers`), então os Gerais, os Raros universais e os de área ou profissão nunca aparecem
  neles. Livro especial novo com gatilho próprio entra no `sealedPlaces`. O peso é o da raridade (`commonEvent`, `rareEvent`,
  `legendaryEvent`) ajustado pelo quanto o evento é específico: evento de profissão (`jobs`) pesa
  `jobEventWeight` por cento e evento universal (sem `places` nem `areas`, os Gerais e boa parte dos Raros)
  pesa `universalEventWeight` por cento, pra que cada atividade mostre mais os eventos dela do que os gerais.
  Livro novo com muito evento universal pede conferir essa proporção de novo. O sorteio tira os
  últimos `eventMemory` que a pessoa viu (`eventos_vistos`) e os últimos `eventChatMemory` do grupo
  (`sceneBoard.recent`, em memória). Se a pessoa já viu tudo, só a memória do grupo vale. Evento com
  `stage` nunca é sorteado: só abre pelo `next` de um desfecho ou por um `plant`.
- **Pendente.** Um por pessoa e conversa (`sceneBoard.pending`, chave `eventKey`), achado também pelo id da
  mensagem e da enquete (`sceneBoard.messages`). Guardado com `KeepGame` (tipo `evento`) e retomado pelo
  `restoreEvent`; o timer de `eventIdle` resolve com o `idle` do evento (ou o texto padrão). O
  `OnModeOff` da economia (`haltEvents`) descarta os pendentes da conversa.
- **Resposta.** O observador `answerEvent` aceita o voto na enquete (só de quem é dono, comparando pelo
  `User` dos ids guardados) ou uma resposta citando a mensagem ou a enquete com o número da opção. O
  `settleEvent` fecha uma vez só (`close`), sorteia o desfecho pelo `weight` mais o `favor` da profissão e
  joga tudo numa transação (`playEvent`): checa se tá viva, cobra o `uses` (item) e o `price` (fixado na
  hora em que o evento abriu, em `prices`) e aplica os efeitos em ordem, parando se a pessoa morrer. Falta de
  item ou de saldo desfaz tudo (`errEventVoid`) e o evento passa batido. Desfecho com `next` abre a etapa na
  hora, com outra enquete.
- **Efeitos** (`eventoefeitos.go`), todos pelas funções de sempre: `earn` (porcentagem do `shiftValue`, o
  ganho médio de um trampo da profissão), `lose` (`burnBalance`, some), `gain`/`forfeit` (itens),
  `findLoot` (o loot da extração, ou os achados da profissão em qualquer outro lugar), `injure` (`hurt`),
  `heal`, `buzz` (fica sob efeito da droga pelo `getHigh` do `mercadonegro.go`, o mesmo do `!usar`, com
  overdose), `nabbed` (cadeia de flagrante pelo `baseJail` com `sentence` e `lockUp`, com a chance de
  escapar do `flagrantOdds`, e solta o refém de quem vai preso), `perish` (`die`), `stain` (`markCrime`),
  `enchant` (`grantCharm` com a própria pessoa de autora; com a `purificacao` ele chama o `purgeCurses` e,
  sem maldição pra tirar, só avisa), `grudge` (maldição que a vítima roga: `grantCharm` com a vítima de
  autora, então o Dízimo e o Pacto pagam pra ela e o `!imovel exorcizar` paga pra ela também; a Casa
  assombrada só pega em quem tem imóvel e, sem casa, o espírito vai embora), `refresh`/`delay` (espera do comando pelo
  `placeCooldown`; onde o comando não tem espera, como o `!mercadonegro`, vale a do `!trabalhar`, e o texto
  diz qual), `breakTool` (ferramenta da extração ou o instrumento de trabalho da área, nunca
  relíquia), `plant` (marca um evento pra depois) `refund` (passa parte do saldo pra vítima) `repealCasino` (revoga o decreto `cassino` positivo do presidente, veja o Cassino furioso) e `fans`/`unfollow` (soma ou corta uma porcentagem dos seguidores
  de todas as redes na tabela `fama`, como o backlash do `!publi`; quem não tem seguidor não ganha nem perde nada). O `grudge` e o `refund` só
  entram em evento cujos lugares todos têm vítima, e o `enchant` não aceita Dízimo, Pacto nem Casa
  assombrada, que precisam de outra pessoa de autora (`offCharms` no teste). Os encantos mais punitivos ficam
  raros: Pacto e Casa assombrada só em evento raro ou lendário (`harshCharms` no teste), e Jejum e Dízimo
  com peso baixo (até uns 20, quase sempre bem menos) no desfecho. Efeito novo é um
  tipo com `apply` que devolve a linha do resumo e com `fortune`, que diz se ele é bom (positivo), ruim
  (negativo) ou neutro (zero). Os `fortune` ficam todos em `buzios.go`, e o compilador cobra o de efeito novo.
- **Búzios encantados** (relíquia passiva, `oracle` na profissão, `buzios.go`). Além do triplo de eventos, a
  enquete de quem tem mostra em cada opção a chance de dar bom e de dar ruim (`choice.reading`, colado no
  texto da opção no `newPendingEvent`, que também serve a etapa do `next`). A chance sai dos mesmos pesos do
  sorteio (`choice.weights`, que o `roll` usa), e cada desfecho conta como bom ou ruim pelo sinal da soma do
  `fortune` dos efeitos dele (`ending.omen`); o que sobra pra 100% é desfecho neutro. Como o voto é achado
  pelo hash do texto da opção e os textos ficam salvos (`labels`), a leitura não atrapalha o voto nem o
  `restoreEvent`. Ela deixa o valor esperado positivo pra quem tem, e é de propósito.
- **Conteúdo.** Cada livro (`eventBook`) é um arquivo `eventos*.go` com nome, `places` padrão e os
  `scenario`; o `eventBooks` em `evento.go` lista todos, e o `!eventos` agrupa o diário pelo nome do livro
  (dois livros com o mesmo nome somam). Texto usa `{@}` pra pessoa e `{alvo}` pra vítima (só em crime com
  vítima), sem travessão. Toda opção tem `label` curta com emoji, a dica de risco (`lowRisk`, `midRisk`,
  `highRisk`) e de 1 a 4 desfechos; todo evento precisa de pelo menos 2 opções sem `needs`, `uses` e `price`.
  Opções com `needs` (`holds`, `inArea`, `isJob`, `armedUp`, `lawman`, `killer`, `stoned`...) são as
  "secretas" de quem tem o item ou a profissão. Evento não dá relíquia, mítico nem luxo; ferramenta e
  Relicário perdido só em evento raro ou lendário. Balanceamento: `earn` de 20 a 300 no evento comum (até
  1500 no lendário), `lose` até 25, `injure` até 50 fora do `highRisk`, `perish` só em opção `highRisk` e
  com peso baixo, e o valor esperado de cada evento perto de zero, pra não mexer na calibração das
  ferramentas de extração.
- **Teste.** O `evento_test.go` confere o catálogo inteiro (ids, itens, encantos, profissões, lugares,
  marcadores, etapas alcançáveis, opções livres) e joga todo desfecho de todo evento num banco de verdade.
  Rode `go test ./internal/economy/ -run TestEvent` depois de mexer em evento.
