---
name: tarefas-agendadas
description: "Rotinas que rodam sozinhas e moderação (!loteria/runLottery, !lembrete/runReminders, !aniversario/runBirthdays, !advertir e boas-vindas b.welcome). Carregue antes de mexer em algum desses ou ao criar uma rotina nova em segundo plano."
user-invocable: false
---

# Tarefas agendadas e moderação

Coisas que o bot faz sozinho, fora do fluxo de uma mensagem.

- A `!loteria` é a exceção: os bilhetes ficam no banco e o sorteio diário roda em
  `runLottery`, que começa pelo `StartJobs` depois de conectar.
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
