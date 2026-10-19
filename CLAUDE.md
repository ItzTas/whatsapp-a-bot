# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# whatsapp-a-bot

Bot de WhatsApp para grupos, em Go, feito sobre o `whatsmeow`. O `main.go` da raiz só monta e liga
as peças; o código fica nos pacotes de `internal/`, um por modo, mais o `core` com o que todos
usam.

## Código

- **Não escreva comentários no código.** Nenhum comentário de linha, de bloco ou de documentação
  (doc comments). Nomes de funções, tipos e variáveis devem explicar o que o código faz.
  Diretivas do compilador, como `//go:embed` e `//go:build`, não são comentários e continuam
  permitidas.
- Identificadores em inglês, inclusive os nomes dos pacotes (`economy`, `games`...); nomes de
  arquivo, de comandos, modos, tabelas/colunas SQL e todo texto mostrado no WhatsApp (respostas,
  `Help`, mensagens de erro em `fmt.Errorf`/logs) em português, num tom informal.

## Comandos

```bash
go build -o whatsapp-a-bot .   # precisa de CGO (go-sqlite3), então gcc instalado
go vet ./...                   # é o único check que o CI roda
go tool air                    # hot reload; o air é declarado como `tool` no go.mod
docker build -t whatsapp-a-bot .
```

- `go test ./...` roda os testes. Os de integração ficam na raiz (`integracao_test.go`) e montam o
  bot inteiro pelo mesmo `assemble` do `main.go`, com bancos SQLite temporários e um WhatsApp falso
  (`bancada_test.go`): o `say` manda uma mensagem pelo `HandleMessage` e devolve o que o bot
  respondeu, e o `restart` sobe outro bot em cima dos mesmos bancos. Erro no log do bot reprova o
  teste. A troca funciona porque o `b.Client` é um `core.Client` (`internal/core/cliente.go`), que
  embute o `*whatsmeow.Client` e manda as chamadas de rede pela interface `Network`; chamada de
  rede nova que o bot passar a usar entra nessa interface.
- Pra testar a versão localmente, gere o arquivo que o CI gera:
  `git log --reverse --no-merges --format='%H%x1f%s%x1f%b%x1e' > internal/core/versoes/commits.txt`.
- O `air` lê o `.env` (configurado em `.air.toml`) e roda o binário a partir de `./tmp/main`.
- O CI (`.github/workflows/build.yml`) roda `go vet` e publica a imagem multi-arch
  (amd64/arm64) em `ghcr.io` a cada push na `main`.
- Os comandos de mídia chamam binários externos em tempo de execução: `ffmpeg`, `yt-dlp`,
  `gallery-dl`, `magick` (ImageMagick com suporte a webp) e `tesseract` (com os dados `por` e
  `eng`, usado pelo `!ocr`). A imagem Docker já instala todos.

## Configuração (variáveis de ambiente)

- `BOT_DONO`: números dos donos, separados por vírgula. Sem ele ninguém consegue usar os
  comandos de dono (`!modo`, `!blacklist`, `!whitelist`, `!economia`).
- `LLM_PROVIDER_{API,URL,KEY,MODEL}`: modelo principal da IA (padrão `anthropic`), com cota por
  hora por pessoa.
- `LLM_LOCAL_{API,URL,KEY,MODEL}`: modelo de reserva compatível com OpenAI (padrão
  `http://localhost:11434/v1`, Ollama). Cada camada só é ativada se `_MODEL` estiver definido.
- `STT_URL`, `STT_MODEL`, `STT_KEY`: transcrição de áudio para o `!transcrever`.

## Arquitetura

### Pacotes

- `internal/core` tem o `Bot` (cliente, mensagens, nomes e apelidos, relacionamentos e adoções,
  modos, filtro e jogos salvos), os ganchos que os módulos registram (`ganchos.go`), os comandos
  de ajuda (`!comandos`, `!modos`, `!feedback`) e os de dono do filtro (`!modo`, `!blacklist`, `!whitelist`).
  Também guarda os utilitários que mais de um módulo usa, um arquivo por assunto: `texto.go`,
  `marcacoes.go`, `grupo.go`, `tempo.go`, `citacoes.go`, `web.go`, `midia.go`, `respostas.go`,
  `contas.go`, `sorteio.go`, `restricoes.go` e `risada.go`.
- `internal/llm` é o cliente da IA (`Assistant`, os modelos, a cota e o `Tool` das ferramentas) e
  não sabe nada do bot.
- O resto é um pacote por modo: `economy`, `romance` (com a família), `games`, `stats`, `jokes`
  (zoeira, palavras e risada), `ai` (os comandos de IA), `meters`, `moderation`, `utilities` e
  `city` (o Cidade Dorme).
  Cada comando fica no pacote do modo da categoria dele (os de ajuda no `core`), e o código de
  apoio fica com quem usa: o `tueconomia.go` é da `economy`, e o `!tu` da `stats` chama o
  `EconomyProfile` dela.
- Quem pode importar quem: todo mundo usa `core` e `llm`; `romance`, `games` e `stats` usam
  `economy`; `jokes` usa `economy` e `romance`; `ai` usa `economy`, `games` e `romance`; `meters`
  usa `ai`. O `core` nunca importa um módulo, e o Go não deixa ter ciclo. Função que mais de um
  módulo precisa vai pro `core` quando não depende de nenhum módulo; se depende, fica no módulo
  dela e quem usa importa ele, dentro dessa lista.
- Cada módulo tem um `bot` próprio (`modulo.go`) que embute o `*core.Bot`, então `b.SendText`,
  `b.Account` e `b.Messages` funcionam direto. Ele guarda os stores e os `sync.Map` do módulo e
  um campo pra cada módulo de que depende (`b.economy.Reward(...)`). O `type Module = bot` é o
  nome que o `main.go` e os outros módulos usam, o `New` monta o módulo e o `Register` entrega os
  comandos e os ganchos pro `core.Bot`.
- Só é exportado o que outro pacote usa. Isso vale também pro método de uma interface do `core`
  implementada num módulo, como o `Vote` das enquetes: método minúsculo de outro pacote não
  satisfaz a interface, e o type assertion falha em silêncio.
- `games`, `stats`, `jokes` e `ai` têm consultas próprias no `mensagens.db`, e por isso embrulham
  o `core.MessageStore` num `messageStore` local (o `b.messages`) com os métodos delas. Consulta
  que mais de um módulo usa vai pro `core`.
- O `//go:embed` só lê pastas do próprio pacote, por isso `migracoes/`, `assets/` e `figurinhas/`
  ficam em `internal/core/` e `palavras/` fica em `internal/games/`.

### Fluxo de uma mensagem

`main.go` abre os bancos, cria o `core.Bot` (`core.New`), monta os módulos na ordem das
dependências e chama o `Register` de cada um nesta ordem: `romance`, `games`, `city`, `meters`,
`stats`, `jokes`, `economy`, `ai`, `moderation` e `utilities`. Os ganchos
(`internal/core/ganchos.go`) rodam na ordem em que foram registrados. Cada `events.Message` vira
uma goroutine em `Bot.HandleMessage` (`internal/core/bot.go`), que segue esta ordem:

1. Os interceptadores da moderação (`AddInterceptor`: o que apaga a mensagem de quem tá mutado e o
   `forgetUnpinned`, que esquece a mensagem fixada que alguém desfixou na mão), os
   comandos de dono (`AddOwnerCommands`: o `filterCommands` do `core` e o `!economia`) e o
   `answerPrivate` (`privado.go`) rodam primeiro e **ignoram o filtro**. No privado, quem tem uma
   resposta pendente (`b.AwaitPrivate`, usado pelo `!jokenpo @pessoa`, pelo
   `!2verdades1mentira`, pelo chute do espião no `!espiao` e pela noite do `!cidadedorme`) tem a
   mensagem tratada pelo jogo. Mensagem
   com `!` só vai pro pedido se ele ficou com aquele comando
   (`b.AwaitPrivate(ids, answer, "matar")`); senão segue o caminho normal. As outras mensagens
   privadas continuam passando pelo filtro.
2. `allowed` aplica blacklist/whitelist de grupos e pessoas. **O modo padrão é `whitelist`**
   (`filterDefault`), então numa instalação nova o bot ignora tudo até o dono liberar.
3. A mensagem é gravada no `mensagens.db` (é dela que saem ranking, medidores, IA, etc.). Depois
   os reatores (`AddReactor`) tentam reagir, e o primeiro que reagir encerra: o `reactIfMarked`
   da `jokes` vem antes do `reactToAilments` da `economy`. O `AddAfterMessage` (o
   `settleMissions`, o `announceAssemblies`, o `announcePayouts` e o `offerTip`) roda no fim, com `defer`.
4. Os observadores (`AddObserver`) recebem a mensagem como possível resposta dos jogos em
   andamento: `answerBmc`, `answerVdd`, `AnswerPoll`, `answerImpostor`, `answerEspiao`,
   `answerForca`, `answerTermo`, `answerGuess`, `answerVelha` e `answerBatata` na `games`, e o
   tribunal, `answerBlackjack`, `answerCrash` e `answerEvent` (a escolha dos eventos) na `economy`.
   `AnswerPoll` conta os votos das
   enquetes do `!quemfalou`, `!quiz`, `!impostor`, `!espiao`, `!processar`/`!processaria`
   (`trialPolls`) e `!2verdades1mentira` (`mentiraPolls`). `answerGuess`
   (`internal/games/adivinha.go`) atende os jogos de "quem mandar a resposta primeiro ganha",
   como o `!completa`, o `!emoji` e o `!bandeira`. O `!impostor` e o `!espiao` dividem a sala
   de jogadores e a enquete que só conta voto de quem joga (`roomPlayer` e `roomBallot`, em
   `internal/games/sala.go`).
5. Sem `!`: as respostas passivas (`AddPassive`) disparam em paralelo as reações automáticas
   (palavras, risada, links e a IA enxerida). Com `!`: os filtros de comando (`AddCommandFilter`)
   podem ficar com a mensagem; senão o comando é procurado nos `AddCommands` de todos os módulos, o
   `enabledCommand` escolhe entre os que têm aquele nome e estão com o modo ligado, e ele só executa
   se nenhuma guarda (`AddGuard`) barrar. A guarda da economia (`economyGuard`) segura quem tá
   morto, preso ou refém. Depois do `Run`, os `AddAfterCommand` recebem o comando que rodou: o
   `stirLive` da economia usa ele pra soltar os eventos ao vivo. Comando que não existe vai pro `AddUnknownCommand` (o `iainventa`). O
   filtro de comando do romance é o `answerProposal` (`consentimento.go`), que vê se o
   `!aceitar`/`!recusar` (ou `!aceito`/`!recuso`) responde um pedido de casamento ou namoro; sem
   pedido pendente, a mensagem segue pro comando da economia. O da economia é o
   `answerCrashCommand` (`crash.go`), que faz o `!sacar` sacar do `!crash` de quem tá no foguete;
   quem não tá segue o caminho normal.

As tarefas em segundo plano (`runLottery`, `runAnimalGame`, `runReminders`, `runBirthdays`, `runTaxes`,
`runAuctions`, `runLoanSeizures`, `runWanted`, `runGravity`, `runSuccession` e `runPins`) entram pelo `AddJob` e começam no `StartJobs`, depois de
conectar. O `*events.GroupInfo` vai pros `AddGroupInfo` (o `welcome` da moderação). Quem sai do grupo vai pros `AddDeparture` (o `unseatDeserter` da economia, que tira o presidente que saiu), chamados pelo `b.Departed`: o `HandleGroupInfo` chama com o `Leave` do evento, e a moderação chama depois do `!ban` e do ban das advertências, porque o WhatsApp não manda `GroupInfo` pra quem removeu, e o bot não ficaria sabendo. O
`announceUpdate` do `core` também é um job, mas roda uma vez só (veja "Versões e novidades").

### Versões e novidades

- A versão sai sozinha dos commits (`internal/core/versao.go`), começando em `v0.0.0` e subindo
  commit a commit pelo tipo do Conventional Commits: `!` ou `BREAKING CHANGE:` no corpo sobem o
  major, `feat` o minor, `docs`, `ci`, `build`, `test`, `style` e `refactor` não mudam nada, e o
  resto (`chore`, `fix`, `perf`...) sobe o patch.
- O CI grava o `git log` cru em `internal/core/versoes/commits.txt` antes do build (por isso o
  checkout usa `fetch-depth: 0`), e o binário embute esse arquivo. Ele fica no `.gitignore`, então
  build local não tem versão e não avisa nada.
- **As novidades vêm do trailer `Novidade:` no corpo do commit**, uma linha por novidade (dá pra
  repetir o trailer), em português e no tom informal das respostas do bot, falando do que muda pra
  quem usa: `Novidade: chegou o !golpe, pra derrubar o presidente na marra`. Commit sem trailer
  muda a versão mas não aparece no aviso, e commit com trailer sobe pelo menos o patch. Todo commit
  que muda algo que a galera vê no WhatsApp leva o trailer; refatoração, docs e ajuste interno não.
- O `PrepareUpdate`, chamado no `main.go` antes de conectar, compara a revisão embutida com a
  última gravada na tabela `versoes` e já separa os grupos com o submodo `anuncios` ligado (e que
  passam no `AllowedIn`). Por isso só recebe o aviso quem ligou o `anuncios` antes da atualização.
  Depois de conectar, o `announceUpdate` manda a lista de versões com novidade (da mais nova pra
  mais velha) e grava a revisão. Na primeira vez, sem nada gravado, mostra só a versão atual.

### Comandos, categorias e modos

- **Todo comando é registrado no `comandos.go` do pacote dele** (`economyCommands`,
  `gamesCommands`...; os de ajuda ficam no `helpCommands` do `core`), que o `Register` do módulo
  passa pro `AddCommands`: nome, aliases, categoria, textos de ajuda (`Help`, que também alimentam
  o `!comandos`) e a função `Run`. Cada comando costuma viver num arquivo próprio com o mesmo nome
  (`internal/romance/casar.go` → `b.casar`).
- As categorias ficam no `internal/core/comandos.go`. O `!comandos` mostra as seções na ordem do
  `menuRank`, e dentro de cada seção os comandos aparecem na ordem do registro.
- Nomes de comando e de modo são comparados sem acento e sem maiúscula (`core.FoldLetters` em
  `Command.Matches` e no `!modo ativar`), então `!abraçar` e `!abracar` caem no mesmo comando.
  Não cadastre um alias que só difere de outro nome pelo acento.
- Cada categoria aponta para um **modo** (`internal/core/modos.go`). Os modos são ligados por
  conversa, ficam na tabela `conversa_modos` e começam todos desligados. Uma categoria com
  `Mode == ""` está sempre disponível.
- Um modo pode ter **submodos** (`submodes` em `chatModes()`). O submodo aparece embaixo do pai no
  `!modos`, é ligado à parte pelo nome dele, fica de fora do `tudo` e só vale quando o pai também tá
  ligado. Pra checar um modo que pode ser submodo, use `b.ChatModeOn` (ou o `modeActive` do `core`
  com a lista de modos ativos), nunca `b.Modes.Enabled` direto. Uma categoria pode apontar pra um
  submodo, como o `!perseguir`, que fica no `iaenxerida`.
- Um modo pode ser **exclusivo** (`exclusive` em `chatModes()`, hoje o `cidadedorme`): ligar ele
  pausa os modos que brigam com ele (os que não estão no `coexists` dele), guardando quem pausou
  em `conversa_modos_pausados`, e desligar religa esses modos. O `tudo` não liga modo exclusivo.
  Quando dois comandos têm o mesmo nome, o `enabledCommand` (`internal/core/bot.go`) fica com o
  do modo ligado e, entre os ligados, com o do modo exclusivo. Módulo que precisa saber quando o
  modo dele sai do ar (desligado, derrubado por um exclusivo ou pausado) registra
  `b.OnModeOff(modo, ...)`, que o `!modo` chama depois de responder.
- Ao criar um comando numa categoria nova, ou mudar o que um modo libera, atualize também as
  descrições em `chatModes()` (`internal/core/modos.go`) e a ajuda do `!modo` no `filterCommands()`
  (`internal/core/filtro.go`). Categoria nova também entra no `menuRank`.
- Respostas a palavras sem `!` ficam em `answerWords` (`internal/jokes/modos.go`), atrás do modo
  `palavras`.

### Estado e persistência

- **Dois bancos SQLite**: `sessao.db` é a sessão do `whatsmeow` (não mexa no schema) e
  `mensagens.db` é o banco do bot.
- O schema do `mensagens.db` vem das migrações `goose` em `internal/core/migracoes/` (embutidas via
  `//go:embed` e aplicadas na inicialização). Mudança de schema = novo arquivo numerado
  `0000N_nome.sql` com `-- +goose Up`/`Down`; nunca edite uma migração existente.
- Cada domínio tem um `*Store` com acesso ao mesmo `*sql.DB`, criado no `New` do módulo dono e
  pendurado no `bot` dele (o `walletStore` é da `economy`, o `quizStore` da `games`). O que mais
  de um módulo usa fica no `core.Bot` (`Relationships`, `Adoptions`, `Nicknames`, `Names` e
  `Modes`).
- As tabelas da economia são por economia, não por grupo: grupos ligados com `!economia vincular`
  dividem as mesmas linhas, e o `ledgerDB` da `economy` troca o `chat` pelo grupo principal em
  toda consulta (skill `economia-base`).
- Estado efêmero de jogos (rodadas de `!bmc`, `!quiz`, `!forca`...) fica em campos `sync.Map` do
  `bot` do módulo e se perde ao reiniciar. Jogos que seguram moedas no meio da rodada (`!blackjack`,
  `!crash`, `!corrida`) descontam a aposta na hora em que ela é feita e pagam no fim. O `!jokenpo`
  não vale moeda nenhuma, nem contra o bot nem no desafio.
- O `mensagens.db` guarda no máximo 2 anos ou 1 milhão de mensagens (`mensagens.go`), com poda
  periódica.
- O `!retrospectiva`, o `!horario` e o `!primeiramsg` leem só o `mensagens.db` e o `risadas`, sem
  tabela própria. As horas usam o fuso `core.BrazilTime`, aplicado no SQL (`statsZoneOffset`). A
  "parceria de papo" sai de mensagens seguidas no chat (até 5 minutos entre elas, via `LAG`),
  porque o banco não guarda resposta nem marcação.

### Identidades

O WhatsApp entrega o remetente como número (`s.whatsapp.net`) ou como LID oculto, às vezes com o
outro em `SenderAlt`. Por isso tabelas guardam `id`/`id_alt` (ou `remetente`/`remetente_alt`) e o
código usa `b.SenderIdentities`/`b.Identities` (`internal/core`) e `ToNonAD()` antes de comparar ou
gravar JIDs. Siga esse padrão em qualquer lógica nova que identifique pessoas.

### Skills de arquitetura

O resto da arquitetura fica em skills em `.claude/skills/`, pra não ocupar contexto o tempo todo.
**Antes de mexer num assunto, carregue a skill dele** (e a `economia-base` junto de qualquer
`economia-*`):

- `economia-base`: carteiras, profissões, deus, `ProfessionOf`, ralos, reset de intervalos.
- `economia-itens`: catálogo, raridades, qualidade, ferramentas, lojas, vendas, mercado negro,
  primordiais, o Meteorito do Bendegó e a horta (`!plantar`, `!colher`, `!horta`).
- `economia-crime`: cadeia, fuga, sequestro, morte, saúde, assalto, suborno, tribunal.
- `economia-presidencia`: presidente, decretos, impeachment, renúncia, golpe.
- `economia-fama`: redes do `!postar`, seguidores, níveis de fama, `!fama` e `!publi`.
- `economia-dinheiro`: leilão, bolão, agiota, empréstimo, ações, imposto, imóveis, empregados,
  títulos, implantes, obras.
- `economia-igrejas`: igrejas dos jogadores, doutrinas, dízimo e cofre.
- `economia-empresas`: empresas de NPC, hoje as de segurança do `!empresa`: contratos, escolta
  contra o alvo fácil, cobertura nos eventos da bolada, reforço e recuperação pelo `!aceitar`.
- `economia-eventos`: eventos com escolha do `!trabalhar`, das extrações e dos crimes, e o `!eventos`.
- `economia-aovivo`: submodo `aovivo`, eventos ao vivo pro grupo todo (corrida do `!pegar`, chefão
  do `!atacar` e enquete coletiva) e o `!aovivo` do deus.
- `encantos-e-zoeiras`: bênçãos e maldições, `!zoar`, `!reagir`, reações de status.
- `romance`: submodo `consentimento`, `!casar`/`!namorar` e conta conjunta.
- `tarefas-agendadas`: loteria, lembretes, aniversários, advertências e boas-vindas.
- `ia-e-midia`: IA, ferramentas dela, mídia e `!buscar`.
- `cidade-dorme`: o modo exclusivo `cidadedorme`, os papéis, as fases e a noite no privado.

Essas skills são a documentação da arquitetura: ao mudar um comportamento descrito numa delas,
atualize a skill no mesmo commit. Assunto novo vai pra skill que combina com ele, ou numa skill
nova listada aqui.
