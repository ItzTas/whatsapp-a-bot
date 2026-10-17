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
  (`!blackjack`, `!crash`, `!corrida`, `!jokenpo @pessoa`) descontam a aposta na hora em que ela
  é feita e pagam no fim. No `!jokenpo`, a aposta do desafiado é descontada quando ele manda a
  jogada no privado.
- O `!leilao` (`leilao.go`) é o leilão do dia: o bot leiloa um item por conversa, e ninguém leiloa
  item da própria mochila. O item sai do `auctionLot`: um rodízio fixo por conversa
  (`auctionRotation`, o `catalog` filtrado pelo `auctionGoods` e embaralhado com a semente do chat,
  andando uma posição por `localDay`), então o item nunca repete de um dia pro outro e só volta
  depois de passar a lista toda. O `auctionGoods` pega épico e lendário que não é ilegal, não é da
  loja e não é caixa de relíquia. Com `auctionRelicChance` (sorteio do `dailyRand`), o dia é de
  relíquia, num rodízio próprio das `relics()`. O lance mínimo é o `auctionOpening`: o `value`, o
  `luxuryPrice` pros `luxuryGoods` (pra não sair mais barato que a `!luxo`) e `auctionRelicPercent`
  (hoje 3000%) do `value` na relíquia, alto de propósito: relíquia não foi feita pra ser comprada, então não baixe esse preço pra ela ficar acessível. O leilão só vira linha na tabela `leiloes` (`chat`, `dia`,
  `item`, `lance`, `licitante`, `fecha_em`) no primeiro lance; antes disso ele é calculado na hora
  (`todaysAuction`). O `bidAuction` desconta o lance, devolve o do coberto e grava tudo numa
  transação só, e lance nos últimos `auctionExtension` empurra o `fecha_em` (que começa na
  meia-noite de BRT), então o leilão de ontem pode seguir aberto uns minutos depois da virada; o
  `loadAuction` pega sempre a linha aberta mais antiga e só cai no de hoje quando não tem nenhuma.
  O `runAuctions`, iniciado junto do `runTaxes`, fecha as linhas vencidas com um
  `DELETE ... RETURNING` (então reiniciar não entrega duas vezes), põe o item na mochila de quem
  ganhou com qualidade normal e anuncia na conversa (respeitando o modo `economia` e o `allowedIn`
  de quem ganhou). Ninguém recebe as moedas do lance vencedor: é ralo. Dia sem lance não deixa
  rastro.
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
  quem empresta; o empréstimo do banco é outro comando, o `!emprestimo` (abaixo). Quem pede
  manda `!agiota @pessoa <valor> [dias]` e o pedido fica em `debtRequests` (um por quem empresta, na
  mesma chave do `itemOffers`, então o `!aceitar` e o `!recusar` atendem os dois) até a pessoa
  responder. Aceito, a dívida vai pra tabela `dividas` (uma por par devedor/credor) e os juros
  compostos são calculados na leitura (`debt.accrued`): `juros` por dia até o `vence_em` e
  `juros_atraso` depois. Com a dívida vencida, o credor pode `!matar`, `!prender`, `!sequestrar` ou
  `!processar` quem deve (`rivalsWithDebt`) mesmo sem a profissão certa e sem `crime_em`; é cobrança, então
  não marca crime, não tem chance de ir preso na hora e o `!prender` não zera o `crime_em` do
  devedor. A imunidade do presidente continua valendo no `!prender`.
  - A profissão `agiota` mexe nesse sistema pelos campos `lend*`, `collect*` e `garnish` da
    profissão de quem empresta. Os juros (`loanRates`, `lendRate`/`lendLateRate` somados no
    `debtRate`/`debtLateRate`) são lidos no pedido, guardados no `debtRequest` (e no save) e
    gravados na linha da `dividas`, então trocar de profissão depois não muda dívida antiga. O
    `collectBonus` soma na chance do `!matar` e do `!sequestrar` de cobrança (`collectionEdge`),
    o `collectEvery` troca o `debtArrestEvery` do `!prender` de cobrança e o `collectVotes` soma
    votos pra culpado no `!processar` por calote. O `garnish` desconta uma porcentagem do que o
    devedor atrasado ganha no `!trabalhar` e paga a dívida na mesma transação (`salaryLien`, lido
    em paralelo no `trabalhar`, e `garnishPay` → `settleDebt`, o mesmo miolo do `repayDebt`); com
    mais de um credor desses, só o de vencimento mais antigo desconta.
- O `!emprestimo` (`emprestimo.go`) é o empréstimo do banco: o bot empresta moedas novas e as
  moedas pagas de volta somem (ralo). Fica na tabela `emprestimos` (refeita na migração `00080`),
  um por pessoa por conversa, e reaproveita o `debt` do `!agiota` sem credor (o `lender` fica
  vazio), então os juros compostos também saem do `debt.accrued`: `loanRate` por dia até o
  `vence_em` e `loanLateRate` depois. O limite (`loanLimit`) é o `max` do `!trabalhar` da
  profissão vezes `loanLimitShifts`, entre `loanLimitFloor` e `loanLimitCap`, então o presidente e
  os imóveis com `payBonus` pegam mais. Com o prazo vencido, o `runLoanSeizures` (iniciado junto do
  `runAuctions`) penhora o saldo a cada `loanSeizeEvery`: o `seizeLoan` empurra o `penhora_em`
  antes de cobrar (então reiniciar não penhora duas vezes), tira até a dívida toda pelo
  `settleLoan` (o mesmo miolo do `!emprestimo pagar`) e anuncia na conversa, respeitando o modo
  `economia` e o `allowedIn` de quem deve. Saldo zerado não gera aviso. O `!emprestimo` não tem
  nada de cobrança na marra: quem empresta é o banco, então não libera `!matar`, `!prender` nem
  nada parecido.
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
  de Dom Pedro, Santo Graal, Baralho do Zé Pelintra, Espelho da Iara, Lamparina do Lobisomem, Foice da
  Comadre Morte e Livro de São Cipriano) fica fora da loja, das atividades, da `!luxo` e das caixas e baús comuns. Ele só sai
  de caixa com `relicOdds` (hoje o Relicário perdido), que no `roll` troca o prêmio por uma
  relíquia aleatória nessa chance, do `!leilao` nos dias de relíquia e do `!criar` e do `!milagre` de deus; depois passa de mão em
  mão, mas nunca pelo `!furtar` (o `theftChoice` tira as relíquias do sorteio e o furto mirado
  numa relíquia é recusado). A caixa com `relicOdds` também não sai de outra caixa e só cai nas extrações (`loot` das
  `activities`, peso 2). Equipamento com `relicLure` (hoje a Bússola do relicário, que sai de
  caixa comum e, com peso 1, das extrações) multiplica esse peso no `equip` (`lureRelics`) e, com `spent`, é gasto na próxima
  extração. O filtro de metal do Ímã tira a caixa de relíquia do sorteio, mesmo com a bússola. O
  `randomItemOf` pula tudo que é `unboxable()` (luxo, ilegal, relíquia, mítico e
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
  todo épico ou acima com `keep` que não é da loja. A Lamparina do Lobisomem é a única que depende
  da hora: o `apply` dela só mexe na profissão quando o `werewolfHour` (0h às 6h em
  `brazilTime`) dá verdadeiro, e aí soma em quase tudo (crimes, com o `killChance` liberando o
  `!matar` pra qualquer um, proteção, fuga, ganho e espera do `!trabalhar`, `haste`, achados,
  `!daily`, poupança, cassino, caixas, refém e `!acoes`). Como o `professionOf` monta a profissão
  a cada comando, o efeito liga e desliga sozinho, sem estado guardado.
- Mítico é outra raridade própria (`mythic`, entre a `legendary` e a `relic`): são itens de
  extração que só servem pra vender (sem `use`, sem `keep`, sem efeito), dois por atividade, com
  peso 1 no `loot`. Eles só saem com relíquia: o `equip` tira os míticos do `loot` quando nenhum
  item equipado naquela extração é relíquia, então a ferramenta relíquia da atividade (ou o Saco
  do Homem do Saco) e a Vela do Negrinho do Pastoreio liberam eles, e o `source` da atividade
  mostra quais relíquias liberam no "Onde conseguir". São `unboxable()`, então não saem de caixa
  nem de baú, mas têm qualidade e podem ser furtados. Na venda pro mercado, o `marketPercentOf`
  paga pelo menos `mythicSalePercent` (hoje 90%) por eles, ou o `salePercent` da profissão se for
  maior; ele é aplicado dentro do `marketPayoutOf` e do `marketPayout`, então vale no `!vender`
  (todas as formas), no valor de mercado da `!mochila` e no `!item`.
  O `!vender inuteis` vende mítico junto com o resto.
- As ferramentas de extração que não são relíquia (rara, épica e lendária de `!pescar`,
  `!minerar` e `!catar`) têm um tempo padrão de se pagar, hoje **135 minutos**, contando o que a
  pessoa recebe vendendo no `!vender`, não o valor cheio dos itens. O tempo é o `value` da
  ferramenta dividido pelo líquido por hora: o valor médio do melhor item entre `luck + 1`
  sorteios do `loot` montado pelo `equip` (item `gradable()` vezes a qualidade média da melhor de
  `grade + 1` qualidades), vezes o `marketPercentOf` do item (o `marketSalePercent`, ou o
  `mythicSalePercent` nos míticos, sem o `salePercent` de profissão), menos `breaks` vezes o `value` cheio da ferramenta (a reposição sai pelo preço da
  loja), vezes os usos por hora da espera. O ajuste é nos atributos do `gear` (`luck`,
  `fewerJunk`, `moreRare`), não no preço nem na espera: o `faster` é o mesmo pra toda ferramenta
  de extração da mesma raridade, hoje 0 na rara (10 minutos, igual à base), 40 na épica (6), 50
  na lendária (5) e 80 na relíquia (2). O `grade` também é fixo por raridade: 1 na rara, 2 na
  épica, 3 na lendária e 4 na relíquia. Ferramenta nova ou mudança no `loot` de uma
  atividade precisa recalibrar as ferramentas dela pra esse tempo. As ferramentas base (Vara de
  pesca, Picareta e Rede de insetos) ficam de fora e se pagam em uns 60 minutos. Como o `loot`
  da mina vale mais, as picaretas têm um `luck` a menos que a vara e a rede da mesma raridade.
  As relíquias de extração que fazem o papel da ferramenta (Vara de Iemanjá, Picareta de Serra
  Pelada, Rede do Curupira e o Saco do Homem do Saco, que é acessório porque o `!vasculhar` não
  tem ferramenta base) seguem a mesma conta com outro tempo, hoje **30 minutos**, usando o `value`
  delas como preço; como nunca quebram, não têm custo de quebra. Os míticos entram nessa conta
  (só elas acham), então mexer em mítico também pede recalibrar as relíquias de extração. A Vela
  do Negrinho do Pastoreio fica de fora, porque só soma em cima da ferramenta que a pessoa já tem.
- O tipo de cada item (`itemCategories` em `itens.go`) não é um campo: é a primeira regra da lista
  que bate com os campos do item (`pet`, `arms`, `heist`, `gift`, o loot das `activities`...), e o
  que não bate em nada vira `Raridades`. O `Contrabando` pega o ilegal que não é arma nem droga, e
  o `Vantagens` pega os itens com `flex` que não são relíquia, luxo nem ilegais. O `!item`, o `!item <tipo>`, a `!mochila` e a `!loja`
  agrupam por ele (`categoryGroups`), com a raridade na bolinha da linha. Item novo cai no tipo
  sozinho; tipo novo é só entrar na lista, na posição certa, porque a ordem decide o empate.
  O `!vender tipo <tipo>` (ou `categoria`) vende tudo o que a pessoa tem daquele tipo, menos os
  ilegais, pelo mesmo `sellBatch` do `!vender #12`. A palavra `tipo` é obrigatória porque vários
  nomes de tipo já acham item sozinhos (`!vender pesca` vende a Vara de pesca).
  As vendas em lote de uma palavra só ficam no mapa do `parseBulkSale`, cada uma com o filtro
  (`pick`) e a dica, e todas passam pelo mesmo `sellBatch` do `!vender #12`:
  - `!vender inuteis` (ou `inutil`, `sobras`): tudo o que não tem `perk()` (o "Pra que serve" do
    `!item`) e não tem `gift`, de qualquer raridade.
  - `!vender bugigangas` (ou `bugiganga`, `tralhas`): tudo o que não tem `effectPerk()`, que é o
    `perk()` sem a parte do `usePerk()` (o `!usar` que só manda texto, com ou sem `keep`). Então
    entram também comida, presente e colecionável. Os bichinhos (`pet`) ficam de fora.
  - `!vender duplicados` (ou `duplicadas`, `repetidos`, `repetidas`): `quantity - 1` de cada item.
    Como o `takeGrades` tira da pior qualidade pra melhor, quem fica na mochila é o de melhor
    qualidade.
  Os ilegais nunca entram. Efeito novo precisa ir pro `effectPerk()`, e não pro `usePerk()`,
  senão o item vira bugiganga (ou inútil) e é vendido.
- O número de cada item (`#12`) é a posição dele no `catalog` (`item.number()`/`itemByNumber`), não
  vai pro banco. Ele aparece na `!mochila` e no `!item <nome>`, e o `!vender #12 #40 #87` (só números,
  todos com `#`, separados por espaço ou vírgula) vende tudo o que a pessoa tem de cada um numa transação só
  (`sellNumbered` → `sellBatch` → `walletStore.sellItems`). Item novo no meio do `catalog` empurra o número de
  todos os que vêm depois dele.
- A qualidade (`qualidade.go`) é a coluna `qualidade` do `inventarios`, que faz parte da chave, então
  cada qualidade de um item é uma linha própria. Os níveis ficam em `qualities` (de `-3` horrível a `3`
  lendária, com o `0` normal), cada um com a porcentagem do `value` que vale (`price`), o selo
  (`badge`) e o peso no sorteio. A horrível e a lendária têm peso bem baixo, então só aparecem por
  sorte ou com muito sorteio extra pro lado delas. Os pesos são montados pra média dar 100%, mas o
  "melhor de N" das ferramentas depende da distribuição inteira, então mexer em porcentagem ou peso
  pede recalibrar as ferramentas.
  Só a extração sorteia qualidade (`rollQuality` no `roll`), e só pra item `gradable()`: sem efeito
  nenhum (nada de `gear`, caixa, baú, relíquia, luxo, ilegal, encanto, `trade`, `heist`...). Todo
  o resto entra como normal pelo `addItems`. O `grade` da ferramenta (`gear`, somado no `equip`) e
  o `craft.grade` da profissão dão sorteios extras de qualidade naquela extração (positivo fica com
  a melhor, negativo com a pior), e o `rotten` (hoje
  só o caçador de bússola, pra ele não lucrar com a espera curta de extração) deixa tudo com a pior
  qualidade (`worstQuality`, a primeira da lista).
  O `loadInventory` junta as linhas numa `stack` por item, com as qualidades em `grades`, então o
  resto do código continua vendo um item só; a `!mochila` mostra uma linha por qualidade. O
  `takeItems`/`takeGrades` sempre tira da pior qualidade pra melhor, e quem passa item de uma
  pessoa pra outra (`!dar`, `!vender @pessoa`, `!furtar`) usa o `moveItems` ou o
  `addGrades` pra levar a mesma qualidade junto. O `sellItems` calcula o pagamento dentro da
  transação, com as qualidades que saíram de fato (`marketSale.percent`, não um valor pronto). Valor da mochila usa
  `stack.worth()`, nunca `quantity * value`.
- As profissões também vivem em código (`professions` em `profissoes.go`, separadas por área) e
  só o `name` vai pra coluna `profissao` da `carteiras`. Nunca renomeie um `name` existente: quem
  tinha a profissão volta pra `bico`. Os efeitos (campos de `profession` em `profissao.go`) são
  aplicados em cada comando que eles afetam (`!roubar`, `!furtar`, `!pescar`, `!vender`,
  `!comprar`, `!poupar`, `!daily`, `!apostar`/`!cassino`, `!acoes`, `!processar`,
  `!prender`, `!fugir`, `!libertar`, `!reviver`, `!denunciar`, `!sequestrar`, `!venderrefem`, `!assalto`, `!usar`, pensão e indenização), sempre lendo a
  profissão com `professionOf`; efeito novo precisa de campo novo, de texto em `perks()` e do
  gancho no comando.
- O `boxLuck` da profissão (hoje só o youtuber de unboxing) dá sorteios extras no `loot` das caixas
  e baús abertos no `!usar`: o `item.roll` sorteia `boxLuck + 1` vezes e fica com a raridade mais
  alta (`lootRarity`). A chance de relíquia (`relicOdds`) e os prêmios em moeda não mudam.
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
  via `easeEscape`), e as moedas somem, menos no `!prender`, em que o `arrest` (com
  `copKeepsFine`) credita a multa em quem mandou prender; a `!fianca` é outra coisa, é o preço de sair antes. Caminho novo que mande alguém pra
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
  `handleMessage` barra qualquer comando da categoria de economia (`holdIfDead`), não só os com
  `barredInJail`. Se der errado, tem `killCaughtChance` de ir preso na hora (mais
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
  `maxHealth`, solta o refém que a pessoa tiver (`freeHostage`) e o enterro queima de
  `deathLossMinPercent` a `deathLossMaxPercent` por cento do saldo (`burnBalance`, sem piso, pra
  não zerar quem tem pouco), e as moedas somem, a não ser que o `funeral` tenha `heir` (a Foice). Hoje chamam ele o `kill` do `!matar`, a overdose do
  `takeDrug` e o `hurt` da saúde zerada. Caminho novo de morte usa o `die` e mostra o `funeralText`.
- A saúde (`saude.go`) fica nas colunas `saude`/`saude_em` da `carteiras`: vale o número guardado
  mais 1 ponto a cada `healthRegenEvery` desde o `saude_em`, até `maxHealth`, calculado na leitura
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
  `gangs` (uma por conversa, efêmera e salva com `keepGame`) contra uma vítima só, marcada por quem
  abre; quem entra depois manda `!quadrilha` (marcar outra pessoa é recusado). Com `gangSize` sai
  na hora; se a `gangWindow` acabar com pelo menos `gangMin`, sai assim mesmo, e com menos é
  desfeita. A espera é a do `!roubar` (`robCooldown()` de cada um), checada ao entrar e carimbada
  pra todo mundo na hora do roubo, junto do `crime_em`. A chance é `robSuccessChance` mais o
  `robBonus` de cada um mais `gangHelp` por pessoa a mais, menos o `guard` da vítima e o toque de
  recolher. O alarme da vítima protege igual ao `!roubar`. Se der certo, a quadrilha leva de
  `gangLootMin` a `gangLootMax` por cento do saldo da vítima, dividido igual, e a parte de cada um cresce o `robHaul` da profissão dele (hoje só o Chapéu de Lampião, o mesmo que aumenta o que leva no `!roubar`); se der errado, cada
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
  alguém pra cadeia precisa checar o `untouchable()` da profissão (presidente ou subornado), ou o
  `slipsAway()` se for flagrante na hora. Tudo some quando o `mandato_ate`
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
  `extracao` (espera do `!pescar`, `!minerar`, `!vasculhar` e `!catar`, e o intervalo dos `!empregados`) e o toque de recolher (`curfew()`,
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
  devolve um `wager` com o `boost` da hora da aposta (`prizeBoost`: o `decree` do decreto e o
  `luck` da profissão, o `casinoLuck` do Baralho do Zé Pelintra), que fica guardado na aposta (e no
  save da rodada, em `Boost` e `Luck`) e é aplicado no pagamento pelo `boost.pay`, com o
  `boost.tag` na resposta. O `boost` também leva o `rescued`, sorteado ali mesmo com o
  `rescueOdds` da profissão (hoje só o Baralho, 20%) e salvo em `Rescued`: na derrota, o
  `boost.rescue` devolve a aposta inteira (no `!crash` e na `!corrida`, quem perdeu recebe pelo
  `payRescues` no fim da rodada). O `wager` vem também
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
- O `!golpe` (`golpe.go`) tira o presidente na marra. O primeiro `!golpe` abre a conspiração
  (`coups`, uma por conversa, efêmera e salva com `keepGame`), presa ao presidente da abertura, e
  quem abriu é o líder (`plotters[0]`). Quem manda `!golpe` entra no golpe e quem manda
  `!golpe defender` entra na defesa (`coupLoyal`), inclusive o presidente; não dá pra trocar de
  lado. A abertura carimba a coluna `golpe_em` da `presidencias`, e o próximo golpe contra o mesmo
  mandato só sai depois de `coupEvery`. Quando a `coupWindow` acaba, o golpe precisa de `coupMin`
  golpistas (sem esse mínimo se deus tiver no golpe). A chance (`coupAttempt.odds`) é `coupChance`
  mais `coupHelp` por golpista a mais, menos `coupHelp` por defensor, mais o `crime` da arma de
  cada golpista, menos o `guard` da arma de cada defensor (`arsenal`), menos o `guard` do
  presidente e o toque de recolher; deus golpista dá 100% e deus defendendo (ou presidente) dá 0%.
  Todo golpista leva `crime_em`. Se der certo, o `seize` troca a linha da `presidencias` pro líder
  com `mandato_ate` de `coupTerm`, `golpista = 1`, `golpe_em` agora e abusos e decretos zerados
  (o `UPDATE` exige o presidente da abertura ainda no cargo, senão falha com `errCoupVacant`); o
  presidente derrubado só perde o cargo. Se der errado, os golpistas passam pelo `jailCrew` do
  `!assalto`. O `inaugurate` zera `golpe_em` e `golpista`.
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
  solta da cadeia, da morte e do cativeiro e limpa a ficha. Também enche a saúde (`saude` e
  `saude_em`). Nunca mexe em saldo, mochila,
  poupança, empréstimo nem presidência. Coluna de espera nova entra na lista do `resetCooldowns`.
- Os ralos de dinheiro existem porque a economia só tinha torneira: o dinheiro gasto neles some
  (nunca vai pra ninguém), a não ser quando o texto diz o contrário. Preço de ralo, quando dá, é
  uma porcentagem do saldo com piso, pra pesar igual pra rico e pobre. Pra cobrar e fazer alguma
  coisa na mesma transação, use `walletStore.spend` (debita com `debitBalance` e chama o resto).
- O imposto (`imposto.go`) roda no `runTaxes`, iniciado junto do `runBirthdays`: a partir de
  `taxHour` (BRT), cada conversa com o modo `economia` ligado é marcada na tabela `impostos`
  (`chat`, `dia`) antes de cobrar, então reiniciar não cobra duas vezes. Cobra as faixas marginais
  de `taxBrackets` (`wealthTax`) mais o IPTU do imóvel, limitado ao saldo, e anuncia na conversa
  (respeitando o `allowedIn` de quem mais pagou). O `!transferir` e a venda do `!vender @pessoa`
  não têm taxa fixa; a única taxa de transferência é a do decreto `pix`.
- Fiança, hospital e resgate (`fianca.go`, `hospital.go`, `resgate.go`) passam pelo `buyExit`: o
  preço sai do saldo de quem tá preso/morto/refém e quem paga pode ser outra pessoa. A saída é um
  `UPDATE`/`DELETE` condicionado ao prazo (`releaseHeld`), e se a pessoa já saiu ele devolve
  `errNotHeld` e a transação desfaz a cobrança. O resgate apaga a linha de `sequestros`, o que
  mela uma venda de refém pendente. O `!hospital` tem `whileDead` no `commands()`, que é o que
  deixa ele passar pelo `holdIfDead`; com a pessoa viva, ele não passa pelo `buyExit` e vira o
  tratamento da saúde (`treat`).
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
  continua valendo, porque quem entrega é a quadrilha. O flagrante na hora (`!sequestrar`, `!matar`,
  `!fugir @pessoa` e o `jailCrew` do `!assalto`) checa o `slipsAway()`, que junta o `untouchable()`
  com o `elusive` do Gorro do Saci; o Gorro só escapa do flagrante, não do `!prender`, do
  `!denunciar` nem da batida do `!mercadonegro`. No `!processar`, o `trialBribes` lê a
  propina dos dois lados na hora da sentença: se só um pagou, o `riggedVerdict` dá a vitória pra
  ele sem olhar a enquete (nem o quórum); se os dois pagaram, o júri decide normalmente.
- Os imóveis vivem em código (`estates` em `imovel.go`) e só o nível vai pra coluna `imovel` da
  `carteiras`. Sobe um nível por vez, pagando o preço cheio, e cada nível dá `guard`,
  `dailyBonus`, `payBonus` (porcentagem a mais no `min`/`max` do `!trabalhar`, aplicada no
  `housed`) e cobra IPTU (`tax`) no imposto diário. Nunca reordene a lista: o nível é o índice.
  O `staff` de cada nível é o número de vagas de empregado (`!empregados`); sem imóvel, nenhuma.
  O `storage` de cada nível é quantos itens cabem guardados em casa (`guardados.go`): o
  `!imovel guardar`, o `!imovel tirar` e o `!imovel guardados` mexem na tabela `guardados`, que tem
  as mesmas colunas do `inventarios` (qualidade na chave e `usos`). As funções de mochila têm uma
  versão com a tabela de parâmetro (`loadStacks`, `takeGradesFrom`, `addGradesTo`), e o `shiftItems`
  passa item entre as duas levando a qualidade e o desgaste (`usos`) junto, pra guardar e tirar não
  zerar o desgaste da ferramenta. Item guardado não tá na mochila, então nada que lê o `inventarios`
  enxerga ele: não é furtado, não é apreendido, não entra nas vendas em lote e não dá efeito nenhum
  (passivas, arma, ferramenta) até voltar pra mochila. O nível do imóvel só sobe, então o espaço
  nunca encolhe embaixo de quem já guardou.
- Os empregados (`empregados.go`) são fictícios e vivem em código (`staff`), e só o `id` vai pra
  coluna `tipo` da tabela `empregados`, uma linha por empregado contratado (cada um com o próprio
  `coletado_em`, então contratar outro do mesmo tipo não herda as horas dos antigos). Cada tipo
  faz uma extração (`activity`) sozinho a cada `every`, ajustado pelo decreto `extracao`
  (`pace`), e guarda até `stock` achados. Nada roda em segundo plano: o `!empregados coletar`
  calcula as rodadas desde o `coletado_em` (`ready`), cobra o `wage` de cada rodada do saldo e
  sorteia o `roll()` da atividade nerfada pelo `shift`; as rodadas que o saldo não paga viram
  greve e se perdem. O `shift` tira do `loot` tudo que é lendário ou acima, `unboxable()` e
  Bússola do relicário (`relicLure`), e o empregado não usa ferramenta, equipamento nem
  profissão (sem quebra, sem machucado). O aprendiz puxa a qualidade pra baixo (`gradeLuck`
  -1) e o veterano escolhe o melhor de dois achados (`luck` 1). O `price` é calibrado pra se
  pagar em **72 horas** produzindo sem parar (o líquido por rodada, com o `wage` em 30% do que
  a rodada rende em média na venda), então mexer no `loot` de uma atividade pede recalibrar os
  empregados dela. Nunca renomeie um `id`: o empregado com o id antigo some.
- Os títulos (`titulos` em `titulo.go`) são diferentes do `!apelido`: cada um tem um dono só por
  conversa (tabela `titulos`, PK `chat`, `titulo`), e o equipado fica na coluna `titulo` da
  `carteiras`, que aparece no `!saldo` e no `!ricos` (`titleBadge`). Tomar de alguém custa
  `titleStealPercent` do que a pessoa pagou e devolve `titleRefundPercent` pra ela (`dethrone`).
  O `claimTitle` confere se o dono ainda é o mesmo e falha com `errTitleChanged` se mudou. Nunca
  renomeie um `id`. Cada título dá uma vantagem leve no campo `bonus` (um `flex`), e só o
  equipado vale: o `standing` lê a coluna `titulo` e o `professionOf` aplica o `titled` antes do
  `charmed`. A vantagem é de um de três jeitos: um efeito pequeno direto na profissão, uma
  maldição pra lançar (`titleCurse`, que soma na `faith` e pega com pelo menos `titleHexChance`)
  ou um tipo de abuso de poder (`titleAbuse`, que soma no `abuses`). O `castersOf` também lista
  os títulos. Título não dá bênção: como dá pra lançar em si mesmo, a bênção vira um efeito
  permanente bem mais forte que o resto. Mantenha as vantagens fracas e crescendo com o preço
  (compare com os itens de luxo que dão o mesmo efeito), porque o título passa de mão em mão.
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
  `arms`), drogas (campo `high`) e o contrabando (o resto, como a Balança adulterada e o Dado
  viciado, que valem pelo `flex`, o Colete à prova de balas e o Alvará de soltura falso).
  O `!usar` do alvará (`alvara.go`) tira da cadeia quem usou ou quem foi marcado, pelo `releaseHeld`
  da fiança, e marca o `crime_em` de quem usou; com chance `forgeryCaught`, o carcereiro percebe e
  a pena aumenta `escapePenalty`. Nos dois casos o alvará é gasto. O estoque do dia sai do `dailyRand` (o mesmo sorteio da
  `!luxo`, com outro sal), e cada item tem um preço do dia entre `blackMarketMinPercent` e
  `blackMarketMaxPercent` do `value`. O receptador (`!mercadonegro vender`) paga
  `blackMarketBuyback` por cento desse preço, e o `!vender` comum recusa item ilegal. Além do estoque
  do dia, a muamba (`hotGear`: ferramenta lendária com `tool` e equipamento com `relicLure`, hoje as
  três ferramentas lendárias de extração e a Bússola do relicário) chega pelo `hotGearArrivals`. As
  ferramentas (`hotTool`) dividem um sorteio só: com `hotToolChance` vem uma delas, sorteada entre
  todas, então nunca aparecem duas no mesmo dia. O resto da muamba tem `hotGearChance` de chegar,
  cada item num sorteio próprio do `dailyRand`. Só tem uma de cada por dia e conversa: o
  `claimHotGear` grava a compra na tabela `muambas` (`chat`, `dia`, `item`, `comprador`) na mesma
  transação da cobrança e falha com `errSoldOut` se alguém já levou; batida também conta como
  vendida, porque a mercadoria se perde. Ela não é ilegal: não é apreendida, o receptador
  não compra e a revenda é no `!vender`. Toda
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
    `drugMixOverdose` à chance de overdose, que mata pelo `die`.
  - O `walletStore.arrest` (`!prender` e `!denunciar`) apreende tudo o que é ilegal da mochila de
    quem vai preso (`seizeContraband`), e a batida também. Os outros caminhos de cadeia não
    apreendem.
- Os bichinhos do `!petshop` (`petshop.go`) são itens normais do `catalog` com `shop` e `pet`: o
  `!loja` pula eles, o `!comprar` compra igual (com a mensagem de levar pra casa) e o `sources`
  aponta pro `!petshop` (`item.store()`). Bicho novo é só entrar no `catalog` com `pet: true`.
- Item com `shop` e `rotating` é oferta do dia (`rotatingStock` em `loja.go`): a `!loja` tira ele da
  lista fixa e mostra só os `rotatingShopSize` sorteados do dia na seção "⏳ Só hoje" (o
  `dailyRand` com o sal `loja`, trocando à meia-noite de BRT), e o `!comprar` recusa quando ele não
  tá no sorteio de hoje. Uma das vagas é sempre de ferramenta de extração (`featuredTool`, hoje as
  três épicas: Vara de titânio, Picareta de nióbio e Rede de náilon), em rodízio pelo dia
  (`localDay`), então cada uma volta em até 3 dias; as outras vagas são sorteadas e podem trazer
  outra ferramenta de bônus. Fora isso é item de loja normal: sai de caixa, paga o decreto `loja` e
  tem o desconto da profissão. Item de loja com `days` (dias da semana, em BRT) só aparece no "Só
  hoje" nesses dias (`weekdayStock`), fora das vagas sorteadas, e o `!comprar` recusa nos outros
  (`onSale`); hoje só a Tele Sena, aos domingos.
- O `resets` (zerar esperas no `!usar`) sempre vem com um `limit` (`limiter` em `itens.go`), que é
  a espera própria entre usos e os textos dela: a `caffeine` (coluna `cafeina_em`) do café, do
  energético e do chimarrão, e o `disguise` (coluna `disfarce_em`) do Kit disfarce, que zera o
  `!roubar`, o `!furtar` e o `!sequestrar`. Item novo de zerar espera escolhe um dos dois ou cria
  outro `limiter`, com a coluna numa migração e no `resetCooldowns`.
- As obras do `!doar` (`doar.go`) seguem `publicWorks` em ordem por conversa (`obras` guarda o
  nível e o arrecadado, `obras_doacoes` quanto cada um deu em cada obra). Depois da última vêm
  monumentos com a meta dobrando (`workAt`). Doação que passa da meta segue pra próxima, e a obra
  pronta homenageia quem mais doou pra ela (`obras_concluidas`). `!doar @pessoa` continua sendo o
  `!dar`.
- As zoeiras pagas (`zoar.go`) ficam em `zoeiras` (`chat`, `alvo`, `tipo`, `autor`, `ate`), uma
  por tipo e alvo, e pagar de novo soma o tempo. O `palhaco` é a reação 🤡 do `reactToAilments`, o `apelido` tranca o `!apelido` (`nicknameLocked`, só quem pagou muda) e a
  `maldicao` corta o ganho do `!trabalhar` pela metade. O `!zoar limpar` apaga as zoeiras da
  própria pessoa.
- O `!reagir @pessoa <emoji>` (`reagir.go`, modo `zoeira`) grava o emoji na tabela `reacoes`
  (`chat`, `id`/`id_alt`, `emoji`), um por pessoa por conversa, e marcar de novo troca o emoji. O
  `reactIfMarked` reage com ele em toda mensagem da pessoa no `handleMessage`; como uma mensagem só
  guarda uma reação do bot, quando ele reage o `reactToAilments` não roda.
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
    trigger foi recriado na migração `00078`) e grava um aviso por encanto em `encantos_avisos`. Por isso vale em
    todo caminho de dinheiro sem gancho nenhum. Um ganho que daria valor zero não gasta o encanto, e
    os créditos do próprio trigger não disparam outro (o SQLite roda sem `recursive_triggers`).
    Encanto novo desse tipo é só usar os mesmos campos; não crie outro trigger no `saldo`, senão um
    dispara o outro. O `handleMessage` chama o `announcePayouts` (`ganhos.go`, com `defer`) em toda
    mensagem de conversa liberada, que apaga os avisos da conversa e manda no grupo, então o que
    cair no `runLottery` ou no `runTaxes` só é avisado na próxima mensagem. Quando a praga volta
    (`hexBackfire` ou espelho), o `landCharm` grava o alvo original como `autor`, então quem jogou
    paga pra ele; o Dízimo em si mesmo (`autor = alvo`) é ignorado pelo trigger. O `purges()` é só
    a `purificacao`.
- O `reactToAilments` (`status.go`), chamado no `handleMessage`, reage em toda mensagem de quem tá
  com algum status ruim, se o modo `economia` tiver ligado. O `walletStore.ailments` lê tudo numa
  consulta só. Como o WhatsApp só aceita uma reação do bot por mensagem, vale o primeiro da lista do
  `ailments.emoji`: morto ⚰️, preso 🔒, refém 🪢, palhaço do `!zoar` 🤡, amaldiçoado (o emoji da
  maldição do `encantos`, ou 🧿 pra `maldicao` do `!zoar`), drogado 😵‍💫 e saúde em até um terço
  do `maxHealth` 🤕. Status novo é só entrar nessa lista, na posição da prioridade.
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
