# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

# whatsapp-a-bot

Bot de WhatsApp para grupos, em Go, feito sobre o `whatsmeow`. Tudo vive num único pacote `main`
na raiz do repositório.

## Código

- **Não escreva comentários no código.** Nenhum comentário de linha, de bloco ou de documentação
  (doc comments). Nomes de funções, tipos e variáveis devem explicar o que o código faz.
  Diretivas do compilador, como `//go:embed` e `//go:build`, não são comentários e continuam
  permitidas.
- Identificadores em inglês; nomes de comandos, modos, tabelas/colunas SQL e todo texto mostrado
  no WhatsApp (respostas, `help`, mensagens de erro em `fmt.Errorf`/logs) em português, num tom
  informal.

## Comandos

```bash
go build -o whatsapp-a-bot .   # precisa de CGO (go-sqlite3), então gcc instalado
go vet ./...                   # é o único check que o CI roda
go tool air                    # hot reload; o air é declarado como `tool` no go.mod
docker build -t whatsapp-a-bot .
```

- Não há testes no repositório.
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

### Fluxo de uma mensagem

`main.go` abre os bancos, monta o `bot` e registra o handler de eventos. Cada `events.Message`
vira uma goroutine em `bot.handleMessage` (`bot.go`), que segue esta ordem:

1. Comandos de dono (`ownerCommands()` em `filtro.go`) rodam primeiro e **ignoram o filtro**.
   Logo depois vem o `answerPrivate` (`privado.go`), que também ignora o filtro: no privado, quem
   tem uma resposta pendente (`b.awaitPrivate`, usado pelo `!jokenpo @pessoa` e pelo
   `!2verdades1mentira`) tem a mensagem tratada pelo jogo. As outras mensagens privadas continuam
   passando pelo filtro.
2. `allowed` aplica blacklist/whitelist de grupos e pessoas. **O modo padrão é `whitelist`**
   (`filterDefault`), então numa instalação nova o bot ignora tudo até o dono liberar.
3. A mensagem é gravada no `mensagens.db` (é dela que saem ranking, medidores, IA, etc.).
4. Os jogos em andamento recebem a mensagem como possível resposta (`answerBmc`, `answerVdd`,
   `answerPoll`, `answerForca`, `answerTermo`, `answerGuess`, `answerVelha`, `answerBatata`,
   `answerBlackjack`, `answerCrash`, `answerImpostor`). `answerPoll` conta os votos das enquetes
   do `!quemfalou`, `!quiz`, `!impostor`, `!processar`/`!processaria` (`trialPolls`) e `!2verdades1mentira`
   (`mentiraPolls`). `answerGuess` (`adivinha.go`) atende os jogos de "quem mandar a resposta
   primeiro ganha", como o `!completa`, o `!emoji` e o `!bandeira`.
5. Sem `!`: `answerModes` dispara as reações automáticas (palavras, risada, links).
   Com `!`: procura o comando em `commands()` e só executa se o modo da categoria estiver ligado.
   Antes disso, o `answerProposal` (`consentimento.go`) vê se o `!aceitar`/`!recusar` (ou
   `!aceito`/`!recuso`) responde um pedido de casamento ou namoro; sem pedido pendente, a mensagem
   segue pro comando da economia.

### Comandos, categorias e modos

- **Todo comando é registrado em `bot.commands()` (`bot.go`)**: nome, aliases, categoria, textos
  de ajuda (`help`, que também alimentam o `!comandos`) e a função `run`. Cada comando costuma
  viver num arquivo próprio com o mesmo nome (`casar.go` → `b.casar`).
- Nomes de comando e de modo são comparados sem acento e sem maiúscula (`foldLetters` em
  `botCommand.matches` e no `!modo ativar`), então `!abraçar` e `!abracar` caem no mesmo comando.
  Não cadastre um alias que só difere de outro nome pelo acento.
- Cada categoria aponta para um **modo** (`modos.go`). Os modos são ligados por conversa, ficam
  na tabela `conversa_modos` e começam todos desligados. Uma categoria com `mode == ""` está
  sempre disponível.
- Um modo pode ter **submodos** (`submodes` em `chatModes()`). O submodo aparece embaixo do pai
  no `!modos`, é ligado à parte pelo nome dele, fica de fora do `tudo` e só vale quando o pai
  também tá ligado. Pra checar um modo que pode ser submodo, use `b.chatModeOn` (ou `modeActive`
  com a lista de modos ativos), nunca `modes.enabled` direto. Uma categoria pode apontar pra um
  submodo, como o `!perseguir`, que fica no `iaenxerida`.
- Ao criar um comando numa categoria nova, ou mudar o que um modo libera, atualize também as
  descrições em `chatModes()` (`modos.go`) e o `help` do `!modo` em `ownerCommands()`.
- Respostas a palavras sem `!` ficam em `answerWords` (`modos.go`), atrás do modo `palavras`.

### Estado e persistência

- **Dois bancos SQLite**: `sessao.db` é a sessão do `whatsmeow` (não mexa no schema) e
  `mensagens.db` é o banco do bot.
- O schema do `mensagens.db` vem das migrações `goose` em `migracoes/` (embutidas via
  `//go:embed` e aplicadas na inicialização). Mudança de schema = novo arquivo numerado
  `0000N_nome.sql` com `-- +goose Up`/`Down`; nunca edite uma migração existente.
- Cada domínio tem um `*Store` com acesso ao mesmo `*sql.DB` (`relationshipStore`,
  `walletStore`, `filterStore`, `modeStore`...), pendurado na struct `bot`.
- Estado efêmero de jogos (rodadas de `!bmc`, `!quiz`, `!forca`...) fica em campos `sync.Map`
  da struct `bot` e se perde ao reiniciar. Jogos que seguram moedas no meio da rodada
  (`!blackjack`, `!crash`, `!corrida`, `!jokenpo @pessoa`, `!leilao`) descontam a aposta na hora em que ela
  é feita e pagam no fim. No `!jokenpo`, a aposta do desafiado é descontada quando ele manda a
  jogada no privado. No `!leilao` (`auctions`), o item sai da mochila ao abrir, o lance de quem
  está ganhando é descontado e devolvido quando alguém cobre, e lance no fim prorroga o prazo; ao
  desligar o bot ele tenta devolver item e lance, mas se o processo morrer de vez o que estava
  preso se perde.
- O `!processaria` (`processaria.go`) e o `!processar` (`processar.go`) dividem o mesmo tribunal:
  o processo aberto fica em `trialChats` (um por conversa, pros dois comandos) e a enquete do júri
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
- O submodo `consentimento` (do `romance`, em `consentimento.go`) troca o `!casar @pessoa` e o
  `!namorar @pessoa` por um pedido em `romanceProposals` (um por pessoa pedida, salvo com o
  `keepGame`), que a pessoa responde com `!aceitar`/`!recusar` em até `proposalWindow`. Com ele
  ligado, a forma `@pessoa1 @pessoa2` do `!casar`/`!namorar` é recusada e o `!terminar @a @b` só
  vale se quem mandou for um dos dois. A relação aceita grava `consentido = 1` em
  `relacionamentos` (um casamento antigo sem consentimento é só oficializado com `confirm`). Com a
  economia ligada, o pedido pode vir com um valor (dote no casamento, presente no namoro), checado
  no pedido e transferido no aceite (`payProposal`; se faltar moeda, o pedido cai), e o casamento
  cobra `weddingPartyCost` de quem pediu (sem saldo, casa no cartório de graça). Só casamento
  consentido dá pensão no `!terminar` e tem `!conjunta` (`conjunta.go`, tabela `contas_conjuntas`,
  chave `pessoa1`/`pessoa2` com as contas canônicas em ordem): qualquer um dos dois deposita e
  saca, e no divórcio o `splitJoint` apaga a linha e divide o saldo meio a meio.
- A `!loteria` é a exceção: os bilhetes ficam no banco e o sorteio diário roda em
  `runLottery`, iniciado no `main.go` depois de conectar.
- Os `!lembrete` também ficam no banco (`lembretes`, com quem criou em `criador`/`criador_alt`)
  e saem pelo `runReminders`, iniciado junto do `runLottery`. Cada lembrete guarda o `proximo`
  envio e é reagendado antes de ser mandado; se o bot ficou fora do ar por mais de
  `reminderGrace`, o envio atrasado é pulado. Na hora de mandar, o lembrete respeita o modo
  `utilidades` e a blacklist/whitelist (`allowedIn`, com o criador no lugar do remetente).
- Os `!aniversario` ficam na tabela `aniversarios` (`pessoa`/`pessoa_alt`, `dia`, `mes`,
  `ultimo_ano`) e saem pelo `runBirthdays`, iniciado junto do `runReminders`. Todo dia a partir
  de `birthdayHour` (BRT) um `UPDATE ... RETURNING` marca o `ultimo_ano` antes de mandar, então
  reiniciar não duplica; se o bot ficou fora na hora, os parabéns saem quando ele voltar no mesmo
  dia. 29/02 vira 28/02 em ano não bissexto. O envio respeita o modo `utilidades` e o `allowedIn`,
  com o aniversariante no lugar do remetente.
- As advertências do `!advertir` ficam em `advertencias`. Ao chegar em `warningLimit` o bot bane
  com o `banAction` do `!ban` (`moderacao.go`) e só zera a ficha se o ban der certo. As
  boas-vindas ficam em `boas_vindas` (uma por grupo) e saem do `b.welcome`, ligado ao
  `*events.GroupInfo` no `main.go`: ele junta as entradas de um grupo por `welcomeDelay` numa
  mensagem só e respeita o modo `moderacao` e o `allowedIn` de quem entrou.
- O `!bolao` fica no banco (`boloes`, `bolao_opcoes`, `bolao_apostas`): a aposta desconta na
  hora e o `!bolao resultado` divide o pote entre quem acertou, proporcional ao que apostou (a
  sobra do arredondamento vai pra maior aposta); se ninguém acertou ou o bolão é cancelado, todo
  mundo recebe de volta. Fechar, dar resultado e cancelar é só pra quem abriu, admin ou dono.
- O `!agiota` (`agiota.go`, com `!emprestar` de alias) é o empréstimo entre pessoas, e o agiota é
  quem empresta; o bot não empresta nada (a tabela antiga `emprestimos` ficou sem uso). Quem pede
  manda `!agiota @pessoa <valor> [dias]` e o pedido fica em `debtRequests` (um por quem empresta, na
  mesma chave do `itemOffers`, então o `!aceitar` e o `!recusar` atendem os dois) até a pessoa
  responder. Aceito, a dívida vai pra tabela `dividas` (uma por par devedor/credor) e os juros
  compostos são calculados na leitura (`debt.accrued`): `juros` por dia até o `vence_em` e
  `juros_atraso` depois. Com a dívida vencida, o credor pode `!matar`, `!prender`, `!sequestrar` ou
  `!processar` quem deve (`rivalsWithDebt`) mesmo sem a profissão certa e sem `crime_em`; é cobrança, então
  não marca crime, não tem chance de ir preso na hora e o `!prender` não zera o `crime_em` do
  devedor. A imunidade do presidente continua valendo no `!prender`.
- As `!acoes` guardam o preço por conversa em `acoes_precos` (os saldos também são por conversa);
  o passeio aleatório avança um passo a cada `stockStepEvery` quando alguém consulta, e as
  posições ficam em `acoes_posicoes`. As empresas vivem em código (`stockCompanies` em
  `acoes.go`) e só o `ticker` vai pro banco: nunca renomeie um ticker existente.
- As tabelas da economia (`carteiras`, `inventarios`, `loteria_bilhetes`, `boloes`...) guardam
  quem é a pessoa em `pessoa`/`jid`, com a chave canônica do `b.account()`, que já junta LID e
  número.
- Os itens da economia ficam na tabela `inventarios`, mas o catálogo vive em código (`catalog`
  em `itens.go`) e só o `id` de cada item vai pro banco. Nunca renomeie um `id` existente: os
  itens com o id antigo somem das mochilas. As propostas do `!vender @pessoa` são efêmeras
  (`itemOffers`) e só trocam item e moedas na hora do `!aceitar`.
- Relíquia é uma raridade própria (`relic`, acima da `legendary`), não uma lendária: ferramenta
  relíquia nunca quebra (`sturdy: 100`), tem `moreRare: 100` e nunca volta de mão vazia (`alwaysFinds`, que tira a entrada vazia do loot no `equip`); ferramenta lendária tem
  `sturdy: 97` e `grace: 100`. Item com `rarity: relic` (hoje a
  Vara de Iemanjá, a Picareta de Serra Pelada, a Rede do Curupira, o Saco do Homem do Saco, a
  Vela do Negrinho do Pastoreio, a Excalibur e as passivas Muiraquitã, Gorro do Saci, Chapéu de Lampião, Escamas do Boitatá, Coroa
  de Dom Pedro e Santo Graal) fica fora da loja, das atividades, da `!luxo` e das caixas e baús comuns. Ele só sai
  de caixa com `relicOdds` (hoje o Relicário perdido), que no `roll` troca o prêmio por uma
  relíquia aleatória nessa chance, e do `!criar` e do `!milagre` de deus; depois passa de mão em
  mão, mas nunca pelo `!furtar` (o `theftChoice` tira as relíquias do sorteio e o furto mirado
  numa relíquia é recusado). A caixa com `relicOdds` também não sai de outra caixa e só cai nas extrações (`loot` das
  `activities`, peso 2). Equipamento com `relicLure` (hoje a Bússola do relicário, que sai de
  caixa comum e, com peso 1, das extrações) multiplica esse peso no `equip` (`lureRelics`) e, com `spent`, é gasto na próxima
  extração. O filtro de metal do Ímã tira a caixa de relíquia do sorteio, mesmo com a bússola. O
  `randomItemOf` pula tudo que é `unboxable()` (luxo, ilegal, relíquia e
  caixa de relíquia). Ferramentas (`gear` com `tool`) não somam entre si: se a pessoa tem mais de
  uma da mesma atividade, o `equip` só aplica a de maior `value` (`bestTool`), e as outras ficam
  paradas na mochila. Os acessórios (`gear` sem `tool`, como isca, capacete e bússola) continuam
  valendo junto com ela. A sorte de acessório com `skipsRelics` (a isca) vai pro `plainLuck` do
  `roll`, e nesses sorteios extras a caixa de relíquia é ignorada. A Excalibur é arma legal: o `bestWeapon` lê toda arma do `catalog` (não só as
  ilegais), mas ela não é apreendida porque o `seizeContraband` só pega item `illegal`. O
  `!vasculhar` não tem ferramenta base, então a relíquia dele (o Saco do Homem do Saco) é acessório,
  sem `tool` e sem `sturdy`. A Vela do Negrinho do Pastoreio também é acessório, mas vale nas
  quatro extrações: não substitui a ferramenta, só soma sorte e tira a mão vazia do sorteio
  (`alwaysFinds`). As relíquias passivas usam o mesmo `flex` dos itens de luxo (o
  `passives` lê qualquer item com `flex`), mas nunca levam `keep`: o `luxuryGoods` põe na `!luxo`
  todo épico ou acima com `keep` que não é da loja.
- As ferramentas de extração que não são relíquia (rara, épica e lendária de `!pescar`,
  `!minerar` e `!catar`) têm um tempo padrão de se pagar, hoje **100 minutos**, contando o que a
  pessoa recebe vendendo no `!vender`, não o valor cheio dos itens. O tempo é o `value` da
  ferramenta dividido pelo líquido por hora: o valor médio do melhor item entre `luck + 1`
  sorteios do `loot` montado pelo `equip`, vezes `marketSalePercent` (sem o `salePercent` de
  profissão), menos `breaks` vezes o `value` cheio da ferramenta (a reposição sai pelo preço da
  loja), vezes os usos por hora da espera. O ajuste é nos atributos do `gear` (`luck`,
  `fewerJunk`, `moreRare`), não no preço nem na espera: o `faster` é o mesmo pra toda ferramenta
  de extração da mesma raridade, hoje 0 na rara (10 minutos, igual à base), 40 na épica (6), 50
  na lendária (5) e 80 na relíquia (2). Ferramenta nova ou mudança no `loot` de uma
  atividade precisa recalibrar as ferramentas dela pra esse tempo. As ferramentas base (Vara de
  pesca, Picareta e Rede de insetos) ficam de fora e se pagam em uns 60 minutos. Como o `loot`
  da mina vale mais, as picaretas têm um `luck` a menos que a vara e a rede da mesma raridade.
  As relíquias de extração que fazem o papel da ferramenta (Vara de Iemanjá, Picareta de Serra
  Pelada, Rede do Curupira e o Saco do Homem do Saco, que é acessório porque o `!vasculhar` não
  tem ferramenta base) seguem a mesma conta com outro tempo, hoje **45 minutos**, usando o `value`
  delas como preço; como nunca quebram, não têm custo de quebra. A Vela do Negrinho do Pastoreio
  fica de fora, porque só soma em cima da ferramenta que a pessoa já tem.
- O tipo de cada item (`itemCategories` em `itens.go`) não é um campo: é a primeira regra da lista
  que bate com os campos do item (`pet`, `arms`, `heist`, `gift`, o loot das `activities`...), e o
  que não bate em nada vira `Raridades`. O `!item`, o `!item <tipo>`, a `!mochila` e a `!loja`
  agrupam por ele (`categoryGroups`), com a raridade na bolinha da linha. Item novo cai no tipo
  sozinho; tipo novo é só entrar na lista, na posição certa, porque a ordem decide o empate.
  O `!vender tipo <tipo>` (ou `categoria`) vende tudo o que a pessoa tem daquele tipo, menos os
  ilegais, pelo mesmo `sellBatch` do `!vender #12`. A palavra `tipo` é obrigatória porque vários
  nomes de tipo já acham item sozinhos (`!vender pesca` vende a Vara de pesca).
- O número de cada item (`#12`) é a posição dele no `catalog` (`item.number()`/`itemByNumber`), não
  vai pro banco. Ele aparece na `!mochila` e no `!item <nome>`, e o `!vender #12 #40 #87` (só números,
  todos com `#`, separados por espaço ou vírgula) vende tudo o que a pessoa tem de cada um numa transação só
  (`sellNumbered` → `sellBatch` → `walletStore.sellItems`). Item novo no meio do `catalog` empurra o número de
  todos os que vêm depois dele.
- As profissões também vivem em código (`professions` em `profissoes.go`, separadas por área) e
  só o `name` vai pra coluna `profissao` da `carteiras`. Nunca renomeie um `name` existente: quem
  tinha a profissão volta pra `bico`. Os efeitos (campos de `profession` em `profissao.go`) são
  aplicados em cada comando que eles afetam (`!roubar`, `!furtar`, `!pescar`, `!vender`,
  `!comprar`, `!poupar`, `!daily`, `!apostar`/`!cassino`, `!acoes`, `!processar`,
  `!prender`, `!fugir`, `!libertar`, `!reviver`, `!denunciar`, `!sequestrar`, `!venderrefem`, `!assalto`, pensão e indenização), sempre lendo a
  profissão com `professionOf`; efeito novo precisa de campo novo, de texto em `perks()` e do
  gancho no comando.
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
- O `!prender` (`prender.go`) só funciona pra profissão com `jailTime` (policial, delegado, juiz),
  e só em quem tentou `!roubar`, `!furtar` ou `!sequestrar` nas últimas `crimeWindow`: a tentativa
  marca o `crime_em` na `carteiras` (`rob`, `stealItem` e `kidnap`, via `markCrime`) e a prisão zera ele, então
  cada crime rende uma prisão só. Toda ida pra cadeia parte do mesmo `baseJail` (o `jailTime`
  das profissões que prendem também é ele), e o `jailFactor` da profissão de quem é preso e o
  decreto de pena somam ou tiram uma porcentagem desse tempo (`sentence`); caminho novo de cadeia
  usa o `baseJail` e não um tempo próprio; o `arrestEvery` é a espera entre prisões
  (coluna `prisao_em`). A saída da cadeia fica em `preso_ate` na `carteiras`. Toda ida pra cadeia
  (`!prender`, `!denunciar`, X-9, pego no `!matar`, no `!sequestrar` e no `!assalto`) cobra uma
  multa de `fineMinPercent` a `fineMaxPercent` por cento do saldo e corta `arrestEaseCut` por cento da espera do `!fugir` que faltava (`lockUp`,
  via `easeEscape`), e as moedas somem; a `!fianca` é outra coisa, é o preço de sair antes. Caminho novo que mande alguém pra
  cadeia precisa chamar o `lockUp` também. Comando com
  `barredInJail` no `commands()` é barrado no `handleMessage` enquanto quem mandou tá preso ou em
  cativeiro (hoje `!apostar`, `!cassino`, `!blackjack`, `!crash`, `!corrida`, `!roubar`, `!furtar`,
  `!sequestrar`, `!venderrefem`, `!matar`, `!assalto`, `!quadrilha` e `!libertar`).
- O `!fugir` (`fugir.go`) tenta tirar quem tá preso da cadeia, com chance `escapeChance` mais o
  `escapeBonus` da profissão (negativo pra polícia), uma tentativa a cada `escapeCooldown`
  (coluna `fuga_em`). Fugir zera o `preso_ate` e marca o `crime_em`, então a polícia pode
  prender de novo; falhar soma `escapePenalty` ao `preso_ate`. Quem não tá preso mas tá em
  cativeiro (vendido ou não) tenta fugir do sequestro, com chance `breakoutChance` mais o
  `breakoutBonus` da profissão (positivo pra polícia, segurança, mágico, chaveiro...), na mesma
  espera `fuga_em`. Dar certo apaga a linha de `sequestros` sem marcar crime; falhar soma
  `breakoutPenalty` ao `ate`. Se a pessoa tá presa e em cativeiro ao mesmo tempo, a cadeia vem
  primeiro.
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
  `crime_em`, então dá pra `!prender` e `!denunciar`. Se der certo, a vítima ganha `morto_ate`
  na `carteiras` por `deathHold` e solta o refém que tiver (`freeHostage`); enquanto tá morta, o
  `handleMessage` barra qualquer comando da categoria de economia (`holdIfDead`), não só os com
  `barredInJail`. Se der errado, tem `killCaughtChance` de ir preso na hora (mais
  `copVictimCaught` se a vítima tem `jailTime`), por `baseJail` ajustado pelo `jailFactor`.
- A `!recompensa @pessoa <valor>` (`recompensa.go`) desconta o valor de quem bota na hora e guarda
  na tabela `recompensas` (`chat`, `alvo`, `autor`, `valor`, `em`), uma linha por autor e alvo
  (botar de novo soma). Quando o `!matar` dá certo, o `kill` chama o `claimBounty` na mesma
  transação: soma tudo que tem na cabeça da vítima, apaga as linhas e credita no matador, então
  vale também pra cobrança do `!agiota` e pro deus. A `!recompensa retirar @pessoa` devolve só a
  parte de quem mandou. Nenhum outro jeito de morrer (overdose) paga a recompensa.
- O `!reviver` (`reviver.go`) tira a morte antes do prazo: só funciona pra profissão com
  `reviveChance` (necromante) ou pro presidente (`presidentRevive` mínimo), em quem tem `morto_ate`
  no futuro, uma tentativa a cada `reviveCooldown` (coluna `ressurreicao_em`). Dar certo zera o
  `morto_ate`; falhar só gasta a espera.
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
  `gangs` (uma por conversa, efêmera e salva com `keepGame`) contra uma vítima só, marcada por quem
  abre; quem entra depois manda `!quadrilha` (marcar outra pessoa é recusado). Com `gangSize` sai
  na hora; se a `gangWindow` acabar com pelo menos `gangMin`, sai assim mesmo, e com menos é
  desfeita. A espera é a do `!roubar` (`robCooldown()` de cada um), checada ao entrar e carimbada
  pra todo mundo na hora do roubo, junto do `crime_em`. A chance é `robSuccessChance` mais o
  `robBonus` de cada um mais `gangHelp` por pessoa a mais, menos o `guard` da vítima e o toque de
  recolher. O alarme da vítima protege igual ao `!roubar`. Se der certo, a quadrilha leva de
  `gangLootMin` a `gangLootMax` por cento do saldo da vítima, dividido igual; se der errado, cada
  um paga de 10% a 20% do próprio saldo pra vítima. Ninguém vai preso na hora.
- A presidência (`presidencia.go`) fica na tabela `presidencias`, uma linha por conversa (`pessoa`/`jid`,
  `mandato_ate`, as cargas de abuso e as colunas dos decretos). O `!votar @pessoa`
  só abre eleição se não tem mandato em andamento; a urna é efêmera (`elections`, uma por conversa),
  fica aberta `electionWindow` e precisa de `electionQuorum` votos, com empate na sorte. Quem ganha
  fica `termLength` no cargo. Os poderes entram pelo `professionOf`, que aplica `presidential()` na
  profissão de quem é presidente (`min`/`max` do `!trabalhar` vezes `presidentPayFactor`, `dailyBonus` somado de `presidentDailyBonus`, mais `guard`, `releaseChance` e `reviveChance` mínimos e, se não for da polícia,
  `jailTime`/`arrestEvery` próprios), então valem sozinhos em todo comando que lê a profissão. O
  presidente tem imunidade: `!prender` e `!denunciar` recusam com o `immunityText`, a chance de ser
  pego no `!sequestrar` e no `!matar` vira zero, o `jailCrew` do `!assalto` pula ele, o `noSnitch`
  não vale pra ele e o `inaugurate` zera o `preso_ate` de quem toma posse. Caminho novo que mande
  alguém pra cadeia precisa checar o `untouchable()` da profissão (presidente ou subornado). Tudo some quando o `mandato_ate`
  passa, porque toda leitura da presidência filtra por ele, menos os decretos depois de uma renúncia
  (`!renunciar`, abaixo).
- O `!decreto` (`decreto.go`) é só do presidente e lê três tabelas em código, cada decreto com os
  nomes que aceita e a coluna dele na `presidencias`: `percentDecrees` (um número guardado na
  coluna, limitado a `lo`/`hi` na leitura; negativo corta e positivo aumenta, como no `trabalho`,
  que vai de `-workDecreeMaxCut` a `+workDecreeMaxRaise` na espera), `toggleDecrees` (liga/desliga, o `!decreto` sem número inverte) e `onceDecrees`
  (até `limit` usos no mandato, contados na coluna pelo `spendDecree`, que roda o efeito na mesma
  transação e desfaz tudo com `errDecreeIdle` quando não tinha o que fazer, sem gastar o uso).
  O menu, a ajuda do `commands()`, o "em vigor" e o "sobrando" do `!presidente` saem dessas
  tabelas, e o `inaugurate` zera todas as colunas (`decreeColumns`). O `professionOf` pendura os
  decretos da conversa na profissão (`job.decrees`), então os ganchos leem `job.decrees.level(...)`
  e mostram `job.decrees.note(...)`: `daily`, `juros` (pontos na taxa do `!poupar`), `pena` (no `sentence`, então vale pra toda ida pra cadeia),
  `extracao` (espera do `!pescar`, `!minerar`, `!vasculhar` e `!catar`) e o toque de recolher (`curfew()`,
  tirado da chance de `!roubar`, `!furtar`, `!sequestrar`, `!matar`, `!assalto` e `!quadrilha`). Quem não lê a
  profissão chama `b.wallets.decrees`: o `trabalho`/`salario` (`workCooldown`/`payFor`), o `pix`
  (taxa de quem recebe no `!transferir`, que vai pro presidente, a não ser que ele esteja
  na transferência), o `iof` (porcentagem dos juros do `!poupar` cobrada no `redeem` e paga pro
  presidente, a não ser que seja ele resgatando; o `!decreto imposto poupanca` cai nele porque o
  `pickDecree` tenta antes as duas primeiras palavras juntas), a `renda` (porcentagem do que o
  `!trabalhar` e o `!daily` pagam, tirada no `work`/`claim` com o `incomeLevy` e paga pro
  presidente, a não ser que seja ele ganhando; o `!decreto imposto trabalho` e o
  `!decreto imposto daily` caem nela pelo mesmo motivo do `iof`), a `loja` (imposto em cima do
  `!comprar` da `!loja` e do `!petshop`, cobrado no `buyItem` junto do preço e pago pro presidente,
  a não ser que seja ele comprando; o `!decreto imposto compra` cai nela pelo mesmo motivo do
  `iof`, e a `!luxo` e o `!mercadonegro` não pagam), a `fianca` (imposto em cima do preço do
  `!fianca`, cobrado de quem paga no `buyExit` pelo `levy` do `exit` e pago pro presidente na
  mesma transação, a não ser que seja ele pagando; o `!decreto imposto fianca` cai nela pelo mesmo
  motivo do `iof`, e o `!hospital` e o `!resgate` não pagam) e
  os três do cassino, lidos no `placeBet`,
  que só os jogos de cassino usam no lugar do `placeStake` (bolão, leilão e jokenpo continuam): o
  `bets` fecha o cassino, o `cassino` muda o lucro de quem ganha e o `apostas` cobra uma taxa que
  vai pro presidente: quem perde (ou empata) paga a porcentagem em cima da aposta e quem ganha paga
  em cima de tudo que levou, aposta inclusa (`onBet`; ele mesmo não paga; o
  `!decreto imposto aposta` cai nele pelo mesmo motivo do `iof`). O `placeBet`
  devolve um `wager` com o `boost` do decreto da hora da aposta, que fica guardado na aposta (e no
  save da rodada) e é aplicado no pagamento pelo `casinoPrize`, com o `prizeTag` na resposta, e
  com o `levy` da taxa, que o `settleBet` (ou o `holdBet`, nos jogos que seguram a aposta) desconta
  junto da aposta, na mesma transação. Nos jogos que seguram a aposta, a taxa em cima da aposta
  sai na hora e fica guardada na aposta (`heldLevy`, também no save da rodada); se ganhar, o
  `payPrize` desconta do prêmio o que falta pra completar a porcentagem de tudo que levou
  (`onPrize`). O `tudo`, o `metade` e o `%` já contam a taxa da aposta
  (`stakeAfterLevy`); o dobrar do `!blackjack` não paga taxa. Os
  de uma vez são `imposto`, `auxilio` (`presidentAid` de moeda nova pra cada carteira da conversa),
  `caixa` (uma `presidentBoxItem`, a caixa misteriosa comum, na mochila de cada carteira da conversa),
  `anistia` (zera `preso_ate` e o `crime_em` dentro da `crimeWindow`), `indulto @pessoa`
  (`presidentPardons` por mandato), `confisco @pessoa` (`presidentSeizePercent` do saldo pro
  presidente) e `acoes <ticker> <pct>` (mexe direto no `acoes_precos`, e a reversão do passeio
  aleatório puxa o preço de volta com o tempo). Decreto novo precisa de coluna numa migração, de
  entrada numa das três tabelas (e no mapa `decrees()`, se for de uma vez) e do gancho no comando.
- O `!impeachment` (`impeachment.go`) tira o presidente antes do fim do mandato. O primeiro voto
  abre a votação (`impeachments`, uma por conversa, efêmera e salva com `keepGame`), presa ao
  presidente da abertura, que fica aberta `impeachmentWindow`. Cada um vota `sim` ou `nao` (sem
  nada é sim) e pode mudar o voto até fechar, com o peso do `ballotWeight` (o voto de deus vale 3).
  Passa com pelo menos `impeachmentQuorum` votos a favor e mais a favor do que contra, e aí o
  `impeach` joga o `mandato_ate` pra agora, então os poderes e os decretos caem sozinhos e o
  `!votar` volta a abrir eleição. A abertura carimba a coluna `impeachment_em` da `presidencias`
  (zerada no `inaugurate`), e o próximo pedido contra o mesmo mandato só sai depois de
  `impeachmentEvery`. Se o presidente da abertura já saiu do cargo quando a votação fecha
  (renúncia ou fim do mandato), o `impeachmentResult` só avisa e não conta nada.
- O `!renunciar` (`renunciar.go`) é o presidente largando o cargo: o `resign` joga o `mandato_ate`
  pra agora e marca `renunciou = 1` na `presidencias`, então os poderes caem na hora e o `!votar`
  volta a abrir eleição, mas os decretos continuam valendo até o próximo `inaugurate`, que zera o
  `renunciou`. Por isso o `walletStore.decrees` e o `presidency` leem a linha com
  `mandato_ate > agora OR renunciou = 1`; com o mandato vencido, o `presidency` devolve `found`
  falso e um `presidency` só com os `decrees` (sem presidente). Os decretos de imposto continuam
  cobrando, mas viram ralo: o `levy` da presidência sai com o `president` vazio, o
  `collected()` do `pixLevy` fica falso, o `payLevy`, o `payPrize` e o `transferWithPix` não
  creditam ninguém, e os textos usam o `pocket()`/`office()` ("pro ralo"). Todo imposto de decreto
  novo monta o `pixLevy` pelo `p.levy(decreto, quem paga...)` e menciona o `levy.payee()`. O
  `!presidente` sem presidente mostra esses decretos que sobraram (`leftoverText`).
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
- O `!economia resetar intervalos` (dono, em `economia.go`) zera as colunas `*_em` de espera da
  `carteiras` (`resetCooldowns`), da conversa toda ou só de quem foi marcado. Ele também zera
  `preso_ate`, `morto_ate` e `crime_em` e apaga a linha de `sequestros` de quem é refém, então
  solta da cadeia, da morte e do cativeiro e limpa a ficha. Nunca mexe em saldo, mochila,
  poupança, empréstimo nem presidência. Coluna de espera nova entra na lista do `resetCooldowns`.
- Os ralos de dinheiro existem porque a economia só tinha torneira: o dinheiro gasto neles some
  (nunca vai pra ninguém), a não ser quando o texto diz o contrário. Preço de ralo, quando dá, é
  uma porcentagem do saldo com piso, pra pesar igual pra rico e pobre. Pra cobrar e fazer alguma
  coisa na mesma transação, use `walletStore.spend` (debita com `debitBalance` e chama o resto).
- O imposto (`imposto.go`) roda no `runTaxes`, iniciado junto do `runBirthdays`: a partir de
  `taxHour` (BRT), cada conversa com o modo `economia` ligado é marcada na tabela `impostos`
  (`chat`, `dia`) antes de cobrar, então reiniciar não cobra duas vezes. Cobra as faixas marginais
  de `taxBrackets` (`wealthTax`) mais o IPTU do imóvel, limitado ao saldo, e anuncia na conversa
  (respeitando o `allowedIn` de quem mais pagou). O `!transferir`, a venda do `!vender @pessoa` e o
  `!leilao` não têm taxa fixa; a única taxa de transferência é a do decreto `pix`.
- Fiança, hospital e resgate (`fianca.go`, `hospital.go`, `resgate.go`) passam pelo `buyExit`: o
  preço sai do saldo de quem tá preso/morto/refém e quem paga pode ser outra pessoa. A saída é um
  `UPDATE`/`DELETE` condicionado ao prazo (`releaseHeld`), e se a pessoa já saiu ele devolve
  `errNotHeld` e a transação desfaz a cobrança. O resgate apaga a linha de `sequestros`, o que
  mela uma venda de refém pendente. O `!hospital` tem `whileDead` no `commands()`, que é o que
  deixa ele passar pelo `holdIfDead`.
- O `professionOf` lê tudo de uma vez (`walletStore.standing`) e aplica, depois do
  `presidential()`, o imóvel (`housed`), o segurança do `!seguranca` (`escorted`, coluna
  `seguranca_ate`) e a maldição do `!zoar` (`hexed`). Efeito novo que mude `guard`, ganho ou
  bônus de alguém por fora da profissão entra nesse mesmo ponto.
- O `!subornar pagar` (`suborno.go`) cobra `bribePercent` do saldo, com piso `bribeFloor`, e as
  moedas somem. Em troca grava `propina_ate` na `carteiras` por `bribeHold` (pagar de novo soma o
  tempo), e o `paidOff` marca `bribed` na profissão. Enquanto vale, o `untouchable()` (que junta
  presidente e subornado) barra a cadeia como a imunidade do presidente: o `arrestShield` recusa o
  `!prender` e o `!denunciar`, a chance de flagrante no `!sequestrar` e no `!matar` e a batida do
  `!mercadonegro` viram zero e o `jailCrew` do `!assalto` pula a pessoa. O X-9 do `noSnitch`
  continua valendo, porque quem entrega é a quadrilha. No `!processar`, o `trialBribes` lê a
  propina dos dois lados na hora da sentença: se só um pagou, o `riggedVerdict` dá a vitória pra
  ele sem olhar a enquete (nem o quórum); se os dois pagaram, o júri decide normalmente.
- Os imóveis vivem em código (`estates` em `imovel.go`) e só o nível vai pra coluna `imovel` da
  `carteiras`. Sobe um nível por vez, pagando o preço cheio, e cada nível dá `guard`,
  `dailyBonus` e cobra IPTU (`tax`) no imposto diário. Nunca reordene a lista: o nível é o índice.
- Os títulos (`titulos` em `titulo.go`) são diferentes do `!apelido`: cada um tem um dono só por
  conversa (tabela `titulos`, PK `chat`, `titulo`), e o equipado fica na coluna `titulo` da
  `carteiras`, que aparece no `!saldo` e no `!ricos` (`titleBadge`). Tomar de alguém custa
  `titleStealPercent` do que a pessoa pagou e devolve `titleRefundPercent` pra ela (`dethrone`).
  O `claimTitle` confere se o dono ainda é o mesmo e falha com `errTitleChanged` se mudou. Nunca
  renomeie um `id`.
- A loja de luxo (`luxo.go`) sorteia `luxuryStockSize` itens por dia e conversa (semente do dia
  em BRT e do chat) entre os `luxuryGoods`: os itens com `luxury` e os épicos/lendários com
  `keep` que a loja comum não vende. O preço é o `value` vezes `luxuryMarkup`, e o `!comprar`
  também aceita item do estoque do dia. Item com `luxury` nunca sai de caixa nem baú
  (`randomItemOf` ignora).
  - Cada item com `luxury` dá uma vantagem diferente. Os que duram têm `flex` (o `apply` mexe na
    profissão) e valem só de estar na mochila: o `walletStore.passives` lê quais a pessoa tem, em
    paralelo no `professionOf`, e o `adorned` aplica todos (cada tipo conta uma vez, a quantidade
    não importa), antes do `charmed`. Os que acabam têm `bless`, o `id` de uma bênção do `charms`:
    o `!usar` gasta o item e grava a bênção em `encantos` com a própria pessoa de autor
    (`savor`, pelo mesmo `grantCharm` do `!abencoar`). Item de luxo novo precisa de um dos dois.
  - A chance de ganhar em dobro no `!trabalhar` não soma: a da profissão, a da Bolsa de grife e a
    da bênção Sorte entram com `max` no `doubleChance`, e só a Muiraquitã (relíquia) soma por
    cima, no campo próprio `relicDouble`. O sorteio e os `perks()` leem o total pelo
    `doubling()`, que devolve zero com o `noDouble` (a maldição Azar), seja qual for a ordem em que
    os encantos foram aplicados. Fonte nova de dobro usa `max`, nunca `+=`.
- O `!mercadonegro` (`mercadonegro.go`) vende os itens com `illegal` do `catalog`: armas (campo
  `arms`) e drogas (campo `high`). O estoque do dia sai do `dailyRand` (o mesmo sorteio da
  `!luxo`, com outro sal), e cada item tem um preço do dia entre `blackMarketMinPercent` e
  `blackMarketMaxPercent` do `value`. O receptador (`!mercadonegro vender`) paga
  `blackMarketBuyback` por cento desse preço, e o `!vender` comum recusa item ilegal. Toda
  negociação passa pelo `walletStore.smuggle`: tira o que a pessoa dá, marca o `crime_em` e
  entrega a mercadoria, ou, com chance `raidOdds()` (zero pro presidente e pro deus), dá batida
  (`jailSmuggler`): cadeia por `baseJail` ajustado, `lockUp`, apreensão e solta o refém. Item
  ilegal nunca sai de caixa nem baú.
  - A arma funciona só de estar na mochila e só a mais cara conta (`bestWeapon`, lida em paralelo
    no `professionOf`). O `armed` soma o `arms` na profissão: `crime` no `robBonus`,
    `theftBonus` e `kidnapBonus`, `kill` no `killChance` (então arma com `kill` libera o `!matar`
    pra qualquer profissão) e `guard` na proteção.
  - A droga é gasta no `!usar` (`takeDrug`) e guarda o efeito nas colunas `droga`/`droga_ate` da
    `carteiras`; o `standing` só devolve a droga enquanto o `droga_ate` não passou, e o `drugged`
    aplica o `high` na profissão. Só um efeito vale por vez: usar outra troca o efeito e soma
    `drugMixOverdose` à chance de overdose, que mata por `deathHold` (`morto_ate`) e solta o refém.
  - O `walletStore.arrest` (`!prender` e `!denunciar`) apreende tudo o que é ilegal da mochila de
    quem vai preso (`seizeContraband`), e a batida também. Os outros caminhos de cadeia não
    apreendem.
- Os bichinhos do `!petshop` (`petshop.go`) são itens normais do `catalog` com `shop` e `pet`: o
  `!loja` pula eles, o `!comprar` compra igual (com a mensagem de levar pra casa) e o `sources`
  aponta pro `!petshop` (`item.store()`). Bicho novo é só entrar no `catalog` com `pet: true`.
- As obras do `!doar` (`doar.go`) seguem `publicWorks` em ordem por conversa (`obras` guarda o
  nível e o arrecadado, `obras_doacoes` quanto cada um deu em cada obra). Depois da última vêm
  monumentos com a meta dobrando (`workAt`). Doação que passa da meta segue pra próxima, e a obra
  pronta homenageia quem mais doou pra ela (`obras_concluidas`). `!doar @pessoa` continua sendo o
  `!dar`.
- As zoeiras pagas (`zoar.go`) ficam em `zoeiras` (`chat`, `alvo`, `tipo`, `autor`, `ate`), uma
  por tipo e alvo, e pagar de novo soma o tempo. O `palhaco` é a reação 🤡 do `reactIfClowned`
  no `handleMessage`, o `apelido` tranca o `!apelido` (`nicknameLocked`, só quem pagou muda) e a
  `maldicao` corta o ganho do `!trabalhar` pela metade. O `!zoar limpar` apaga as zoeiras da
  própria pessoa.
- As bênçãos e maldições (`encantos.go`) vivem em código (`charms`, com `curse` separando as duas)
  e só o `id` vai pra tabela `encantos` (`chat`, `alvo`, `tipo`, `autor`, `ate`), uma por tipo e
  alvo; lançar de novo só renova o prazo (`MAX`), não soma. Nunca renomeie um `id`. O efeito é o
  `apply`, aplicado no `professionOf` pelo `charmed`, por último, então vale em todo comando que lê
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
- O `mensagens.db` guarda no máximo 2 anos ou 1 milhão de mensagens (`mensagens.go`), com poda
  periódica.
- O `!retrospectiva`, o `!horario` e o `!primeiramsg` leem só o `mensagens.db` e o `risadas`, sem
  tabela própria. As horas usam o fuso `brazilTime`, aplicado no SQL (`statsZoneOffset`). A
  "parceria de papo" sai de mensagens seguidas no chat (até 5 minutos entre elas, via `LAG`),
  porque o banco não guarda resposta nem marcação.

### Identidades

O WhatsApp entrega o remetente como número (`s.whatsapp.net`) ou como LID oculto, às vezes com o
outro em `SenderAlt`. Por isso tabelas guardam `id`/`id_alt` (ou `remetente`/`remetente_alt`) e
o código usa `senderIdentities`/`identities` (`filtro.go`) e `ToNonAD()` antes de comparar ou
gravar JIDs. Siga esse padrão em qualquer lógica nova que identifique pessoas.

### IA e mídia

- `assistant` (`ia.go`) tenta as camadas em ordem (provedor com cota, depois o modelo local);
  `chatModel` (`llm.go`) abstrai Anthropic SDK e APIs compatíveis com OpenAI.
- Toda chamada à IA passa um `iaAsker` (`b.iaAsker(ctx, chat, user)`). Com o submodo
  `iasemlimite` (do `ia`) ligado na conversa, a cota por hora não conta e o modelo local libera a
  vaga sem a pausa de `localRest`.
- A IA tem memória por conversa (`iamemoria.go`, tabela `ia_memorias`): as ferramentas
  `guardar_memoria`, `buscar_memoria` e `esquecer_memoria` entram no `iaToolkit` de toda chamada
  com ferramentas. Cada conversa guarda no máximo `iaMemoryLimit` anotações de até
  `iaMemoryMaxRunes` caracteres; com a memória cheia, a IA precisa apagar uma antes de guardar.
- As ferramentas de sorte (`iasorte.go`: `rolar_dados`, `numero_aleatorio`, `cara_ou_coroa`,
  `sortear`, `embaralhar` e `sortear_membros`) também entram em toda chamada com ferramentas, pra
  IA não inventar resultado de dado ou sorteio. O `!processaria` usa só elas (`randomTools()`), e o
  prompt pede pra decidir pelo menos uma coisa do julgamento na sorte.
- As ferramentas de economia (`iaeconomia.go`: `economia_da_pessoa` e `mais_ricos`) também entram
  em toda chamada com ferramentas. Elas mostram saldo, posição no ranking, poupança, cadeia e a
  profissão (com as vantagens, via `details()`) de quem tá no grupo, e recusam com o modo
  `economia` desligado na conversa.
- Dois submodos do `ia` deixam a IA se meter sem ser chamada. No `iainventa` (`iainventa.go`), um
  `!comando` que não existe (e não é de dono) vai pra IA, que responde como se ele existisse, na
  cota de quem mandou. No `iaenxerida` (`iaenxerida.go`), as pessoas marcadas no `!perseguir`
  (tabela `perseguidos`) têm toda mensagem sem `!` respondida pela IA, numa cota única por
  conversa (`perseguir:<chat>`); se a IA falhar ali, o bot fica quieto. Os dois passam por
  `b.iaAnswer`, o mesmo caminho do `!ia`, com um trecho extra no prompt explicando a situação.
- `b.personTexts` (`iapessoa.go`) busca o nome e as mensagens de uma pessoa pros comandos de IA
  sobre gente (`askAboutPerson`, `!processaria`, `!fofoca @pessoa`, `!ficia`).
- O `!cotacao` e o `!clima` (`cotacao.go`, `clima.go`) leem APIs públicas sem chave com o
  `net/http` padrão e timeout (`fetchPublicJSON`).
- Trabalho pesado com binários externos passa por `withMediaJob` (`utilidades.go`): limita a 2
  jobs simultâneos, aplica timeout, roda com `nice` e reage com ⏳/✅/❌ na mensagem.
- GIFs (`assets/<nome>/*.mp4`) e figurinhas (`figurinhas/`) são embutidos no binário.
- O `!buscar` (`buscar.go`) lê as páginas de busca do Buscapé e da KaBuM (JSON do
  `__NEXT_DATA__`) e da Amazon (HTML, pulando os patrocinados `AdHolder`) com o `net/http`
  padrão, em paralelo, e tenta de novo uma vez quando a loja responde 503/429. Cada loja vira
  uma mensagem: as fotos dos produtos lado a lado (montadas com `magick montage`, sem texto,
  porque a imagem Docker não tem fontes) e a lista na legenda; se as fotos falharem, vai só o
  texto. OLX, Mercado
  Livre, Magalu, Shopee e Facebook Marketplace foram testados e bloqueiam o bot (Cloudflare,
  antibot ou login), então não adianta tentar raspar eles direto.
