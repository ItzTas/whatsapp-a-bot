---
name: commitar
description: "Commita tudo como a /commit global (mesmas flags: wip/quick/fast, fastest, detailed, verbose, push, pr, merge, force), mais a regra do projeto: um trailer Novidade: pra cada coisa nova que a galera vê no WhatsApp."
argument-hint: [wip|quick|fast|fastest|quickest] [detailed|verbose] [push] [pr] [merge] [force]
allowed-tools: Bash(git*), Bash(gh*), Bash(glab*), Read, Glob, Grep, Bash(~/.claude/skills/commit/scripts/*), Bash(~/.claude/skills/pr/scripts/*), Bash(~/.claude/scripts/forge-checks.sh*)
---

# Commit do whatsapp-a-bot

Flags recebidas: `$ARGUMENTS`

**Siga a skill global inteira**, `~/.claude/skills/commit/SKILL.md` (leia ela primeiro): flags,
pedidos extras antes do commit, commits atômicos, Conventional Commits, push, PR/MR, merge e
resumo funcionam igual. Nada de `Co-Authored-By` nem atribuição nenhuma. O que muda neste projeto
é só o que está abaixo, e vale **em todos os modos**, inclusive no `fastest`.

## O trailer `Novidade:`

As novidades que o bot anuncia nos grupos com o submodo `anuncios` saem do corpo dos commits
(`releaseNotes` em `internal/core/versao.go`). Commit sem trailer não aparece no aviso.

- **Uma linha `Novidade:` pra cada coisa nova**, não uma por commit. Comando novo, subcomando,
  item, evento, decreto, mudança de valor, de regra ou de texto que a galera percebe no WhatsApp:
  cada um ganha a sua linha. Se o commit trouxe três coisas que quem joga vai notar, são três
  linhas `Novidade:` (o trailer pode repetir).
- **Quem leva**: todo commit que muda algo visível no WhatsApp, inclusive `fix` que muda
  comportamento que a galera vê. **Quem não leva**: refatoração, docs, skills, testes, CI, ajuste
  interno sem efeito pra quem usa.
- **Uma linha só, sem quebra.** O parser lê linha a linha, então a novidade inteira fica na mesma
  linha, por mais comprida que seja. Não aplique a quebra de ~72 colunas nela, senão o resto some
  do aviso.
- **Formato exato**: `Novidade: ` no começo da linha (com N maiúsculo e os dois-pontos), depois o
  texto. As linhas ficam juntas no fim do corpo, depois de uma linha em branco. No `fastest` e
  em qualquer commit sem corpo, o corpo passa a ser só os trailers.
- **Texto**: em português, no tom informal das respostas do bot, falando do que muda pra quem usa
  (não do código). Cite os comandos com `!`, diga o que dá pra fazer e o que mudou, sem nome de
  função, tabela ou arquivo. Exemplos do histórico:

  ```
  Novidade: chegou o !prestigio, pra quem tem muito patrimônio largar tudo, renascer e ganhar um bônus eterno no !trabalhar e no !daily (com ⭐ no nome)
  Novidade: o toque de recolher do !decreto agora dura só 15 minutos e não atrapalha mais o !golpe
  Novidade: o !daily agora paga de 1000 a 2000 moedas, bem mais que antes
  ```

- **Divisão atômica**: a novidade vai no commit que introduz a coisa. Se você separar um
  arquivo em dois commits, cada um leva as novidades da parte dele.

## Tipo do commit e versão

A versão sobe sozinha pelo tipo (`internal/core/versao.go`): `!`/`BREAKING CHANGE:` sobe o major,
`feat` o minor, `docs`/`ci`/`build`/`test`/`style`/`refactor` não mudam nada e o resto sobe o
patch. Commit com `Novidade:` sobe pelo menos o patch, mesmo sendo `refactor` ou `docs`. Por
isso, não use `refactor`/`docs` pra esconder mudança visível: se tem novidade, o tipo é outro
(normalmente `chore` ou `fix`; `feat` continua raro, como na regra global).

## Antes de commitar

- Se a mudança mexeu num assunto descrito numa skill de `.claude/skills/` e a skill não foi
  atualizada, avise (o `CLAUDE.md` pede skill e código no mesmo commit). Não edite a skill por
  conta própria dentro do `/commitar`.
- No resumo final, liste as linhas `Novidade:` de cada commit junto da mensagem, pra eu revisar
  o que vai aparecer no aviso dos grupos.
