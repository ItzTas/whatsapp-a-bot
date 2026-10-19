---
name: economia-aovivo
description: "Eventos ao vivo da economia (submodo aovivo, motor em aovivo.go, gatilho stirLive pelo AddAfterCommand, jogadores do dia e chance que sobe com o tempo em liveBoard, formatos corrida do !pegar em aovivocorrida.go, chefão do !atacar em aovivochefao.go e enquete coletiva em aovivoenquete.go, catálogos aovivocorridas.go/aovivochefoes.go/aovivoenquetes.go, !aovivo do deus, KeepGame dos tipos aovivo e aovivo-grupo, haltLive). Carregue antes de mexer num evento ao vivo, criar um novo ou mudar o gatilho."
user-invocable: false
---

# Economia: eventos ao vivo

Evento que aparece sozinho no grupo, fica aberto alguns minutos e qualquer pessoa pode entrar. É diferente
do evento com escolha (skill `economia-eventos`), que é de uma pessoa só. Tudo fica atrás do submodo
`aovivo` (`core.ModeLive`, filho da `economia`), e os comandos `!pegar`, `!atacar` e `!aovivo` são da
`core.LiveCategory` (mesmo título da economia, modo `aovivo`). O `economyGuard` trata essa categoria como
da economia: morto não joga, e os dois comandos de jogar têm `BarredInJail`.

- **Gatilho.** O `stirLive` entra pelo `AddAfterCommand` do `core` e só olha comando da categoria com modo
  `economia` num grupo. Cada um anota a pessoa na lista de jogadores do dia (`liveBoard.cue`, quem usou
  comando da economia nas últimas `livePlayerSpan`) e sorteia com o `liveOdds`: zero até `liveRest` depois do
  último evento, sobe em linha reta até `livePeak` e vira certeza em `liveDue`. O modo só é lido quando o
  sorteio passa, e o evento só abre entre `liveFirstHour` e `liveLastHour` (BRT) e se não tiver outro rolando
  na conversa (`liveBoard.events`, um por chat). A noite não conta no relógio (`liveIdle`): na abertura do
  dia ele tá no fim do `liveRest`, como se tivesse acabado de sair um evento, pra não ter evento garantido
  toda manhã. A lista de jogadores do dia corta quem passou de `livePlayerSpan` em toda leitura (`roster`). O relógio, a lista e os últimos `liveMemory` eventos do
  grupo ficam salvos com `KeepGame` do tipo `aovivo-grupo` (no máximo a cada `liveSaveEvery`, ou quando
  entra gente nova), pra reiniciar o bot não soltar evento na hora nem esquecer quem jogou.
- **Formato.** O `rules()` de cada `liveFormat` diz o peso, o mínimo de jogadores do dia (`crowd`), o prazo e
  as funções de montar, mostrar e encerrar. Com menos de `liveCrowd` jogadores do dia só sai corrida. O
  `pickLive` sorteia o formato pelo peso e depois o cenário pela raridade (`eventRarity`), tirando os
  últimos do grupo enquanto der. Chefão e enquete marcam os jogadores do dia no aviso (`liveCall`).
- **Corrida** (`grabFormat`). Quem mandar `!pegar` primeiro fecha o evento (`close`) e leva um desfecho
  sorteado pelo peso. Cenário com `riddle` tem senha: a conta do `arithmetic`, a palavra do `scrambled` ou a
  contagem de emoji do `counting`, e só vale `!pegar <resposta>` certa (`sameAnswer` ignora acento, caixa
  e espaço). Resposta errada leva reação ❌. Sem ninguém em `grabWindow`, sai o texto de `missed`.
- **Chefão** (`bossFormat`). A vida é `toughness` vezes os jogadores do dia (mínimo `liveCrowd`). Cada
  `!atacar` tira de `bossHitMin` a `bossHitMax` (`bossArmed` por cento com arma na profissão, crítico em
  dobro), uma vez a cada `bossRest` por pessoa (`strike`, sob o lock do evento). Depois de cada golpe o
  chefão revida com chance `counter.weight` por cento. Quem zera a vida fecha o evento e todo mundo que
  bateu leva o `victory` com o ganho pesado pela parte do dano (`shareOf`, de `bossShareMin` a
  `bossShareMax` por cento do normal), e quem mais bateu ainda leva o `trophy`. Se `bossWindow` passar,
  quem bateu leva o `defeat`.
- **Enquete coletiva** (`voteFormat`). A enquete fica aberta `voteWindow` e os votos chegam pelo
  observador `answerLiveVote`, que acha a conta canônica de quem votou (`b.Account`), então LID e número
  são uma cédula só, e o voto pode mudar até o fim. A cédula guarda o texto da opção (`label`), e a
  apuração acha a escolha pelo texto (`choiceOf`), pra um deploy que mude a ordem das opções não trocar o
  voto. A opção mais votada ganha (`majority`, empate sorteado entre as empatadas), o desfecho é sorteado
  uma vez só e vale pra todo mundo que votou, cada um com o ganho da própria profissão. Quem tá preso ou em
  cativeiro na apuração fica de fora (`heldNotes`), igual ao `BarredInJail` dos outros dois comandos.
- **Efeitos.** São os mesmos `effect` do `eventoefeitos.go`, aplicados por pessoa no `playLive` (lê as
  profissões em paralelo e roda um `playEvent` por pessoa, uma transação cada). Se a transação de alguém
  falhar, o erro vai pro log, os outros recebem do mesmo jeito e a mensagem mostra o `liveFailure` pra
  quem ficou sem. O `place` fica vazio, então
  `refresh`/`delay` mexem no `!trabalhar`. Não use `breakTool` nem `refund`. O ganho é porcentagem do
  `shiftValue` de quem recebe, igual nos eventos com escolha.
- **Deus.** O `!aovivo [corrida|chefão|enquete]` (`invocar`, só com `divine`) abre um evento na hora, sem
  esperar o relógio, sem olhar o horário nem o mínimo de jogadores, e reinicia o relógio do grupo.
- **Estado.** Se o aviso não sai (`show` falha), o evento é fechado e apagado na hora. O evento em
  andamento é salvo com `KeepGame` do tipo `aovivo` (chave = chat) a cada mudança
  (`keepLive`, que não salva evento fechado) e volta pelo `restoreLive`, que religa o timer. Desligar a
  `economia` ou o `aovivo` (ou ser pausado pelo `cidadedorme`) descarta o evento sem pagar ninguém
  (`haltLive` no `OnModeOff`).
- **Conteúdo.** Texto de cenário segue as regras dos eventos: sem travessão, `{@}` só onde tem uma pessoa
  (o vencedor da corrida, quem bateu no golpe e no contra-ataque, quem mais bateu na vitória do chefão).
  Abertura, fuga do chefão e tudo da enquete são coletivos e não marcam ninguém. Opção de enquete não tem
  `needs`, `uses` nem `price`, e `perish` só aparece em opção `highRisk`, em até 10% das vezes. Prêmio segue
  o `checkPrize`: nada mítico, de luxo ou relíquia, e ferramenta só em cenário raro ou lendário.
- **Teste.** O `aovivo_test.go` confere os três catálogos, joga todo desfecho num banco de verdade e cobre
  a chance, a lista de jogadores, o sorteio, o golpe do chefão, a divisão do prêmio e a maioria. Rode
  `go test ./internal/economy/ -run 'TestLive|TestPickLive|TestBoss|TestMajority|TestSameAnswer'` depois de
  mexer em evento ao vivo.
