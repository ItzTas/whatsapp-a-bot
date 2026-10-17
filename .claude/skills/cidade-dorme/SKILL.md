---
name: cidade-dorme
description: "Modo exclusivo cidadedorme (internal/city: !cidadedorme, !matar/!salvar/!investigar no privado, !votar de dia, papéis em cityRoles, fases em driveCity, cityGame em partida.go, salvo com KeepGame em salvo.go, haltCity no OnModeOff). Carregue antes de mexer no Cidade Dorme ou num comando que sobrescreve outro."
user-invocable: false
---

# Cidade Dorme

O jogo Mafia/Lobisomem, no pacote `internal/city`, atrás do modo exclusivo `cidadedorme`.

- O modo é exclusivo (`ModeCity` em `chatModes()`, com `coexists` só `moderacao` e `utilidades`):
  ligar ele pausa o resto, inclusive a economia. Por isso o `!matar` e o `!votar` do `city` usam o
  mesmo nome dos da economia sem brigar: o `enabledCommand` fica com o do modo ligado e, se os dois
  estiverem ligados, com o do modo exclusivo. Comando novo do jogo com nome repetido segue o mesmo
  caminho, sem alias especial.
- A partida é uma por conversa, em `b.games` (`cityGame`, `partida.go`), e o estado inteiro passa
  pelo `syncCity` (`salvo.go`), que salva com o `KeepGame` (tipo `cidadedorme`) ou apaga com o
  `DropGame` quando a fase vira `cityOver`. O `restoreCity` volta a partida na fase em que ela
  tava, com o mesmo prazo. Campo novo no `cityGame` precisa entrar no `citySave`.
- Quem anda com a partida é o `driveCity` (`cidadedorme.go`), uma goroutine que roda o passo da
  fase atual (`closeCityLobby`, `passCityNight`, `passCityDay`) até a partida acabar. Cada passo
  espera o prazo da fase (`cityDurations`) ou o `wake` (sala lotada, `!cidadedorme começar`, todo
  mundo agiu de noite, todo mundo votou de dia, cancelamento), resolve a fase e já abre a próxima.
  Por isso o restore só precisa chamar o passo da fase salva. A partida acaba sozinha com
  `cityTimeout` ou depois de `cityIdleLimit` rodadas paradas (noite sem escolha do assassino e dia
  sem voto).
- Os papéis ficam no `cityRoles`, com o comando da noite (`command`) e os textos. O `cityCast`
  monta a lista (1 assassino até 6 jogadores, 2 de 7 a 10, 3 acima, mais um médico e um detetive)
  e o `deal` embaralha os papéis, não os jogadores, porque o número da lista é público. As regras
  de alvo de cada papel ficam no `refusal` e a gravação no `choose`, os dois com um mapa por papel.
  Papel novo precisa de entrada no `cityRoles`, no `refusal`, no `choose` (`records`) e no `acted`.
- A noite é no privado: o `awaitCityNight` registra um `AwaitPrivate` por papel vivo com o nome do
  comando dele, e o `answerPrivate` do `core` deixa esse pedido ficar com o `!matar 3` (ou só o
  número) mandado no privado, antes do filtro. Os pedidos ficam em `game.prompts` (um por
  assento), que só a goroutine do `driveCity` mexe, e são soltos no fim da noite.
- O menu da noite é uma enquete no privado (`sendCityPolls`, opções `3. Ana` montadas pelo
  `ballots` a partir dos `targets`, nomes do `cityNames`). O `SendPrivatePoll` do `core` manda a
  enquete e o `prompt.WatchPoll` liga ela ao pedido (um pedido vigia quantas enquetes precisar):
  o `answerPrivate` decifra o voto e entrega o texto da opção pra mesma resposta do `!matar 3`, e
  o `cityNumber` lê o número dela. Se alguma enquete não sair, vai o menu em texto (`menu`). As
  enquetes da noite ficam em `game.polls` (uma lista por assento, salva no `citySave`), e o
  `awaitCityNight` religa elas aos pedidos depois de um restore. Voto numa enquete de outra
  noite é ignorado pelo pedido.
- A enquete do WhatsApp tem no máximo `cityPollLimit` (12) opções. Com mais alvos (`split`), o
  `ballots` divide a lista em enquetes equilibradas ("lista 1 de 2") e a primeira escolha da
  noite trava: o `choose` recusa troca quando `split` e `acted`, venha ela de enquete ou de
  número. O `changeRule` dá o texto da regra (troca livre ou primeira escolha) pra pergunta da
  enquete, pro `menuHint` e pro "Anotado". O assassino pode trocar a escolha
  até a noite acabar; com mais de um, vale a mais escolhida (empate na sorte) e os parceiros são
  avisados no privado. O médico não salva a mesma pessoa duas noites seguidas (`lastSaved`).
- No grupo, o `!matar`, o `!salvar` e o `!investigar` só explicam que é no privado
  (`nightCommand`). O `!votar` (`votar.go`) aceita marcação, o número da lista ou `ninguem`; o
  mais votado sai, e empate (inclusive com o `ninguem`) não elimina ninguém. Todo morto tem o
  papel revelado, e o fim da partida revela todos.
- Desligar o `cidadedorme` (ou ligar um modo que derruba ele) chama o `haltCity` pelo
  `OnModeOff` do `core`, que encerra a partida e avisa no grupo.
- Não tem moeda: a economia nunca tá ligada junto.
