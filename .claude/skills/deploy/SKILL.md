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
  commit. O workflow `build.yml` publica `linux/amd64` e `linux/arm64`.
- O container se chama `whatsapp-a-bot` e roda com `--restart unless-stopped`.
- O volume `whatsapp-a-bot-data`, montado em `/data`, guarda o `sessao.db` (a sessão do WhatsApp)
  e o `mensagens.db`. Ele sobrevive à recriação do container.
- As variáveis de ambiente ficam em `/etc/whatsapp-a-bot.env` no servidor, legível só pelo root.
  **Nunca leia, mostre nem copie o conteúdo desse arquivo**: ele tem chaves e números de
  telefone. Para conferir quais variáveis existem, liste só os nomes:
  `ssudo -H pi -s "sed -n 's/^\([A-Za-z_]*\)=.*/\1/p' /etc/whatsapp-a-bot.env"`.

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
4. Baixa `ghcr.io/itztas/whatsapp-a-bot:sha-<sha>` no servidor, recria o container com o mesmo
   env e o mesmo volume e apaga as imagens antigas do bot.
5. Espera até 60 s (`DEPLOY_LOGIN_TIMEOUT`) pelo `Successfully authenticated` nos logs.

## Depois do script

- **Saída 0**: diga qual tag foi publicada e que o login foi confirmado.
- **Falhou no CI**: mostre o comando `gh run view <id> --log-failed` que o script imprimiu e ofereça
  investigar a falha.
- **Sem login no tempo limite**: mostre as linhas de log que o script imprimiu. Se a sessão caiu,
  quem precisa escanear o QR code é o usuário, com `ssudo -H pi docker logs -f whatsapp-a-bot` no
  terminal. O container continua rodando, então não desfaça o deploy por conta própria.
- **Rollback**, só se o usuário pedir: recrie o container apontando para a tag `sha-` anterior,
  com os mesmos `--env-file`, `-v` e `--restart` que o script usa.
