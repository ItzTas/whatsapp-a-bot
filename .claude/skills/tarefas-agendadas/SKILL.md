---
name: tarefas-agendadas
description: "Rotinas que rodam sozinhas e moderação (!loteria/runLottery, !bicho/runAnimalGame, !lembrete/runReminders, !aniversario/runBirthdays, !advertir e boas-vindas b.welcome, !fixar/!desfixar e runPins, Campo gravitacional do Bendegó runGravity, sorteio do presidente runSuccession). Carregue antes de mexer em algum desses ou ao criar uma rotina nova em segundo plano."
user-invocable: false
---

# Tarefas agendadas e moderação

Coisas que o bot faz sozinho, fora do fluxo de uma mensagem.

- O sorteio do presidente também é rotina: o `runSuccession`, a cada `core.ReminderTick`, abre a
  vaga de quem ficou sem presidente e sorteia um quando o prazo pra eleger acaba, anunciando pelo
  `announce` (skill `economia-presidencia`).
- O Campo gravitacional do Meteorito do Bendegó também é rotina: o `runGravity`, a cada
  `core.ReminderTick`, paga o acumulado da `campo_gravitacional` que tá esperando há
  `gravityEvery` (6h) e anuncia pelo `announce` (skill `economia-itens`).
- A `!loteria` é a exceção: os bilhetes ficam no banco e o sorteio diário roda em
  `runLottery`, que começa pelo `StartJobs` depois de conectar. O resultado sai pelo `announce`
  da economia: em todo grupo vinculado com o submodo `anuncios` ligado (skill `economia-base`).
- O `!bicho` segue o mesmo caminho: as apostas ficam no banco e o `runAnimalGame`, iniciado junto
  do `runLottery`, sorteia as extrações do dia e paga quem acertou, anunciando pelo mesmo
  `announce` (skill `economia-dinheiro`).
- Os `!lembrete` também ficam no banco (`lembretes`, com quem criou em `criador`/`criador_alt`)
  e saem pelo `runReminders`, iniciado junto do `runLottery`. Cada lembrete guarda o `proximo`
  envio e é reagendado antes de ser mandado; se o bot ficou fora do ar por mais de
  `reminderGrace`, o envio atrasado é pulado. Na hora de mandar, o lembrete respeita o modo
  `utilidades` e a blacklist/whitelist (`AllowedIn`, com o criador no lugar do remetente).
- Os `!aniversario` ficam na tabela `aniversarios` (`pessoa`/`pessoa_alt`, `dia`, `mes`,
  `ultimo_ano`) e saem pelo `runBirthdays`, iniciado junto do `runReminders`. A cada
  `core.ReminderTick`, um `UPDATE ... RETURNING` pega os aniversários do dia (BRT) e marca o
  `ultimo_ano` antes de mandar, então os parabéns saem logo depois da meia-noite e reiniciar não
  duplica; se o bot ficou fora na virada, eles saem quando ele voltar no mesmo dia. 29/02 vira 28/02
  em ano não bissexto. O envio respeita o modo `utilidades` e o `AllowedIn`, com o aniversariante no
  lugar do remetente.
- As advertências do `!advertir` ficam em `advertencias`. Ao chegar em `warningLimit` o bot bane
  com o `banAction` do `!ban` (`moderacao.go`) e só zera a ficha se o ban der certo. As
  boas-vindas ficam em `boas_vindas` (uma por grupo) e saem do `b.welcome`, ligado ao
  `*events.GroupInfo` pelo `AddGroupInfo`: ele junta as entradas de um grupo por `welcomeDelay` numa
  mensagem só e respeita o modo `moderacao` e o `AllowedIn` de quem entrou.
- O `!fixar` (`internal/moderation/fixar.go`, só admin ou o dono do bot) manda um
  `PinInChatMessage` com a duração em `MessageContextInfo.MessageAddOnDurationInSecs`. O WhatsApp só
  aceita 24h, 7 dias ou 30 dias (`whatsappPinLength`), então o bot fixa pelo menor prazo que cobre o
  tempo pedido e guarda a mensagem em `mensagens_fixadas` (`ate` = fim, 0 = pra sempre;
  `renovar_em` = próxima ação). O `runPins`, a cada `core.ReminderTick`, pega o que venceu o
  `renovar_em`: se passou do `ate`, desfixa e apaga a linha; senão refixa e reagenda
  (`pin.scheduled`, que renova `pinRenewMargin` antes do WhatsApp soltar e põe o `renovar_em` no
  `ate` quando o fim cai antes disso). Refixar recusado pelo WhatsApp apaga a linha; erro de rede
  tenta de novo em `pinRetry`. Sem responder nada, `!fixar <texto>` manda o texto e fixa a mensagem
  do bot. O `!desfixar` e o interceptador `forgetUnpinned` (alguém desfixou na mão) tiram a linha.
  Se no grupo só admin mexe nas configurações (`IsLocked`), o bot precisa ser admin.
