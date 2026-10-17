---
name: romance
description: "Submodo consentimento do romance (consentimento.go, pedido de !casar/!namorar com !aceitar/!recusar, dote e presente, weddingPartyCost, pensão e !conjunta). Carregue antes de mexer em casamento, namoro ou conta conjunta."
user-invocable: false
---

# Romance: consentimento

Pedidos de casamento e namoro com consentimento.

- O submodo `consentimento` (do `romance`, em `consentimento.go`) troca o `!casar @pessoa` e o
  `!namorar @pessoa` por um pedido em `romanceProposals` (um por pessoa pedida, salvo com o
  `KeepGame`), que a pessoa responde com `!aceitar`/`!recusar` em até `proposalWindow`. Com ele
  ligado, a forma `@pessoa1 @pessoa2` do `!casar`/`!namorar` é recusada e o `!terminar @a @b` só
  vale se quem mandou for um dos dois. A relação aceita grava `consentido = 1` em
  `relacionamentos` (um casamento antigo sem consentimento é só oficializado com `Confirm`). Com a
  economia ligada, o pedido pode vir com um valor (dote no casamento, presente no namoro), checado
  no pedido e transferido no aceite (`payProposal`; se faltar moeda, o pedido cai), e o casamento
  cobra `weddingPartyCost` de quem pediu (sem saldo, casa no cartório de graça). Só casamento
  consentido dá pensão no `!terminar` e tem `!conjunta` (`conjunta.go`, tabela `contas_conjuntas`,
  chave `pessoa1`/`pessoa2` com as contas canônicas em ordem): qualquer um dos dois deposita e
  saca, e no divórcio o `splitJoint` apaga a linha e divide o saldo meio a meio.
