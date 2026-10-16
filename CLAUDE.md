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
  `gallery-dl` e `magick` (ImageMagick com suporte a webp). A imagem Docker já instala todos.

## Configuração (variáveis de ambiente)

- `BOT_DONO`: números dos donos, separados por vírgula. Sem ele ninguém consegue usar os
  comandos de dono (`!modo`, `!blacklist`, `!whitelist`).
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
2. `allowed` aplica blacklist/whitelist de grupos e pessoas. **O modo padrão é `whitelist`**
   (`filterDefault`), então numa instalação nova o bot ignora tudo até o dono liberar.
3. A mensagem é gravada no `mensagens.db` (é dela que saem ranking, medidores, IA, etc.).
4. Os jogos em andamento recebem a mensagem como possível resposta (`answerBmc`, `answerVdd`,
   `answerPoll`, `answerForca`).
5. Sem `!`: `answerModes` dispara as reações automáticas (palavras, risada, links).
   Com `!`: procura o comando em `commands()` e só executa se o modo da categoria estiver ligado.

### Comandos, categorias e modos

- **Todo comando é registrado em `bot.commands()` (`bot.go`)**: nome, aliases, categoria, textos
  de ajuda (`help`, que também alimentam o `!comandos`) e a função `run`. Cada comando costuma
  viver num arquivo próprio com o mesmo nome (`casar.go` → `b.casar`).
- Cada categoria aponta para um **modo** (`modos.go`). Os modos são ligados por conversa, ficam
  na tabela `conversa_modos` e começam todos desligados. Uma categoria com `mode == ""` está
  sempre disponível.
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
  da struct `bot` e se perde ao reiniciar.
- O `mensagens.db` guarda no máximo 2 anos ou 1 milhão de mensagens (`mensagens.go`), com poda
  periódica.

### Identidades

O WhatsApp entrega o remetente como número (`s.whatsapp.net`) ou como LID oculto, às vezes com o
outro em `SenderAlt`. Por isso tabelas guardam `id`/`id_alt` (ou `remetente`/`remetente_alt`) e
o código usa `senderIdentities`/`identities` (`filtro.go`) e `ToNonAD()` antes de comparar ou
gravar JIDs. Siga esse padrão em qualquer lógica nova que identifique pessoas.

### IA e mídia

- `assistant` (`ia.go`) tenta as camadas em ordem (provedor com cota, depois o modelo local);
  `chatModel` (`llm.go`) abstrai Anthropic SDK e APIs compatíveis com OpenAI.
- Trabalho pesado com binários externos passa por `withMediaJob` (`utilidades.go`): limita a 2
  jobs simultâneos, aplica timeout, roda com `nice` e reage com ⏳/✅/❌ na mensagem.
- GIFs (`assets/<nome>/*.mp4`) e figurinhas (`figurinhas/`) são embutidos no binário.
