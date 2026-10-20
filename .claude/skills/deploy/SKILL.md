---
name: deploy
description: Publica no servidor a versão da main que o CI já buildou, recria o container do bot e confirma o login no WhatsApp
argument-hint: "[logs | status]"
allowed-tools: Bash(git*), Bash(gh*), Bash(ssudo*), Bash(.claude/skills/deploy/scripts/*)
---

Coloca no servidor a imagem que o CI gerou para o commit mais recente da `origin/main` e confere
que o bot voltou a conversar com o WhatsApp.

## Como o servidor está montado

- O servidor é acessado pelo alias SSH definido em `DEPLOY_HOST` (padrão `pi`). Os comandos com
  root passam pelo `ssudo -H <host>`, que pede a senha do sudo num prompt gráfico.
- A imagem vem do ghcr: `ghcr.io/itztas/whatsapp-a-bot`, com uma tag `sha-<7 caracteres>` por
  commit e a `latest`, que acompanha a main. O workflow `build.yml` publica `linux/amd64` e
  `linux/arm64`.
- O bot roda pela stack do Compose em `/opt/stacks/whatsapp-a-bot` (variável `DEPLOY_STACK`), que
  o Arcane (`http://192.168.100.10:3552`) também gerencia. O `compose.yaml` usa a imagem
  `ghcr.io/itztas/whatsapp-a-bot:${BOT_TAG:-latest}`; o container se chama `whatsapp-a-bot` e roda
  com `restart: unless-stopped`. Não recrie o container com `docker run`, senão ele sai do
  controle do Compose e do Arcane.
- O volume `whatsapp-a-bot-data` (declarado como `external` no compose), montado em `/data`,
  guarda o `sessao.db` (a sessão do WhatsApp) e o `mensagens.db`. Ele sobrevive à recriação do
  container.
- As variáveis de ambiente ficam no `.env` da stack (`/opt/stacks/whatsapp-a-bot/.env`), que o
  `compose.yaml` passa pro container com `env_file: .env` e que o Arcane mostra e edita na página
  do projeto. O arquivo é do usuário `65532` (o do Arcane), com permissão `600`. O `BOT_TAG` do
  rollback também fica nele. **Nunca leia, mostre nem copie o conteúdo desse arquivo**: ele tem
  chaves e números de telefone. Para conferir quais variáveis existem, liste só os nomes:
  `ssudo -H pi -s "sed -n 's/^\([A-Za-z_]*\)=.*/\1/p' /opt/stacks/whatsapp-a-bot/.env"`.

## Argumento (`$ARGUMENTS`)

- **sem argumento**: faz o deploy (seção abaixo).
- **`logs`**: mostra as últimas linhas do bot, `ssudo -H pi docker logs --tail 50 whatsapp-a-bot`,
  e não muda nada.
- **`status`**: mostra estado, imagem e reinícios do container com
  `ssudo -H pi -s "docker ps -a --filter name=whatsapp-a-bot --format '{{.Status}} {{.Image}}'"`, e
  não muda nada.

## Deploy

Rode, a partir da raiz do repositório:

```bash
.claude/skills/deploy/scripts/deploy.sh
```

O script, em ordem:

1. Faz `git fetch` e pega o sha da `origin/main`. **Só vai para o servidor o que já foi pushado**:
   se houver commits locais que ainda não subiram, avise antes de rodar.
2. Recusa o deploy se o bot estiver rodando nesta máquina (`tmp/main` do `air` ou o binário
   `whatsapp-a-bot`). Duas instâncias com a mesma sessão fazem o WhatsApp derrubar uma delas.
3. Espera o build do CI desse sha terminar (`gh run watch`). Se o build falhar, para sem mexer no
   servidor. Um build pode levar uns 20 minutos por causa da emulação arm64, então rode o script
   em background e espere a notificação.
4. Tira o `BOT_TAG` do `.env` da stack, baixa a `sha-<sha>` e a `latest` no servidor e confere
   se são a mesma imagem. Se a `latest` já for de um build mais novo da main, para sem mexer no
   container e pede pra rodar o deploy de novo. Se forem iguais, recria o container com
   `docker compose up -d` e apaga as imagens antigas do bot.
5. Espera até 60 s (`DEPLOY_LOGIN_TIMEOUT`) pelo `Successfully authenticated` nos logs.

## Depois do script

- **Saída 0**: diga qual tag foi publicada e que o login foi confirmado.
- **Falhou no CI**: mostre o comando `gh run view <id> --log-failed` que o script imprimiu e ofereça
  investigar a falha.
- **Sem login no tempo limite**: mostre as linhas de log que o script imprimiu. Se a sessão caiu,
  quem precisa escanear o QR code é o usuário, com `ssudo -H pi docker logs -f whatsapp-a-bot` no
  terminal. O container continua rodando, então não desfaça o deploy por conta própria.
- **Rollback**, só se o usuário pedir: ponha `BOT_TAG=sha-<anterior>` no `.env` da stack
  (`/opt/stacks/whatsapp-a-bot/.env`) e rode `docker compose up -d` nessa pasta. O próximo deploy
  apaga o `BOT_TAG` e volta pra `latest`.
