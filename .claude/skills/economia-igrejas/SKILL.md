---
name: economia-igrejas
description: "Igrejas da economia (igreja.go e igrejacofre.go, !igreja fundar/entrar/sair/lista/doutrinas/ofertar/reformar/dizimo/milagre/desviar, doutrinas em código, templos e a força da doutrina, tabela igrejas, fiéis em carteiras.igreja, dízimo payTithe no !trabalhar e no !daily, cofre, desvio como crime, sucessão do líder). Carregue antes de mexer em igreja, doutrina, templo, dízimo ou cofre."
user-invocable: false
---

# Economia: igrejas

- As igrejas (`igreja.go`) são fundadas pelos jogadores: o nome é livre e a mecânica vem da
  doutrina, que vive em código (`doctrines`). Só o `id` da doutrina vai pra coluna `doutrina` da
  tabela `igrejas`. Nunca renomeie um `id`: a igreja perde o efeito e o dízimo. Não tem religião
  de verdade no código de propósito, porque dar bônus diferente pra cada religião real é
  ranquear religião entre amigos.
- A tabela `igrejas` tem `id`, `chat`, `chave`, `nome`, `doutrina`, `lider`, `cofre` e
  `fundada_em`, com `chave` única por conversa (o `normalizeItemName` do nome, então nome que só
  muda acento ou pontuação conta como o mesmo). Quem é fiel fica na `carteiras`: `igreja` (o `id`,
  0 sem igreja) e `igreja_desde`. É uma igreja por pessoa por conversa.
- O `!igreja fundar <doutrina> <nome>` é só pra profissão da `areaFe` ou pro deus (`divine`),
  checado no comando pelo `ProfessionOf`. Custa `churchFoundPercent` do saldo com piso
  `churchFoundFloor`, e as moedas somem. Quem funda vira o `lider` e já entra como fiel. O
  `!igreja entrar <nome>` acha a igreja pelo nome (`matchChurch`: exato, começo ou pedaço, igual
  ao `findTitle`).
- Fundar e entrar passam pelo `conversionCooldown` (coluna `conversao_em`, que tá no
  `resetCooldowns`). Sair não tem espera, mas como ela é carimbada na entrada, quem sai não entra
  em outra antes de ela acabar. Fundar ou entrar com uma igreja já larga a antiga.
- Toda saída passa pelo `leaveFlock`: zera `igreja` e `igreja_desde` e, se era o líder, o
  `handOver` passa a liderança pra quem tem o `igreja_desde` mais antigo. Se não sobra ninguém, a
  igreja é apagada e o cofre some. Mudar de profissão não tira ninguém da liderança.
- A doutrina vale pelo `devoted`, aplicado no `ProfessionOf` depois do `titled` e antes do
  `charmed`; o `standing` lê o nome, a doutrina e o templo com `LEFT JOIN igrejas`. Ela aparece no
  `perks()` pelo `churchPerks`. Os efeitos são fracos de propósito, porque a filiação não acaba
  sozinha (compare com os títulos e com as bênçãos, que duram horas).
- O efeito de cada doutrina cresce com o templo: o `effect` e o `apply` recebem a `power` do
  templo (em porcentagem, 100 no primeiro nível) e calculam cada número com o `scaled`. Doutrina
  nova precisa dos dois usando o mesmo número base, senão o texto mente sobre o efeito.
- O dízimo é por igreja, na coluna `dizimo` da `igrejas` (em porcentagem). Ela começa `NULL`, que
  vale o `tithe` da doutrina (o padrão); as consultas leem `COALESCE(i.dizimo, unsetTithe)` e o
  `titheOr` da doutrina resolve. O líder muda com `!igreja dizimo <porcentagem>` (`setTithe`, de 0
  a `churchTitheMax`), e o valor resolvido chega no `ProfessionOf` como `churchTithe`, pro
  `churchPerks`. O dízimo sai só do `!trabalhar` e do `!daily`. O
  `payTithe` lê a igreja dentro da transação do `work`/`claim` e credita o cofre, e o saldo
  entra já líquido (o ganho menos o imposto de renda do decreto e menos o dízimo, os dois em cima
  do bruto). Por isso o trigger `encantos_no_ganho` e a penhora do `garnish` agem em cima do
  líquido. Não faça o dízimo por trigger: o `saldo` não pode ganhar outro trigger.
- O cofre fica fora da `carteiras`, então não paga o imposto do `runTaxes`. Ele enche pelo dízimo
  e pelo `!igreja ofertar` (qualquer fiel tira do próprio saldo e põe no cofre), e só sai por três
  caminhos do líder (`igrejacofre.go`), todos descontando com o `drainCoffer`, que falha com
  `errShortCoffer` quando o cofre não tem o valor.
- Os templos vivem em código (`temples`), e só o nível vai pra coluna `templo` da `igrejas`, então
  nunca reordene a lista: o nível é o índice. O `!igreja reformar` sobe um nível pagando o `cost`
  do próximo com o cofre, e as moedas somem. Cada nível tem a `power` da doutrina.
- O `!igreja milagre <valor>` reparte o valor em partes iguais entre os fiéis menos o líder (a
  sobra da divisão fica no cofre). O líder fica de fora de propósito: numa igreja só com ele, o
  milagre viraria um desvio sem risco.
- O `!igreja desviar <valor>` põe moedas do cofre no saldo do líder. É crime e passa pelo caminho
  normal de cadeia: espera `embezzleCooldown` (coluna `desvio_em`, no `resetCooldowns`, ajustada
  pelo `pace`), recusa com o líder preso ou refém (`errLockedUp`), e a chance de flagrante é
  `embezzleCaughtBase` mais `embezzleCaughtScale` vezes a fração do cofre que tirou (zero pra quem
  `slipsAway()` e pro deus). Sem flagrante, marca o `crime_em`. Com flagrante, o dinheiro fica no
  cofre, a pessoa vai pra cadeia por `baseJail` ajustado pelo `sentence`, com o `lockUp`, e perde a
  liderança pelo `depose`: o `crownSuccessor` pula ela e passa pra quem tem o `igreja_desde` mais
  antigo, e o `igreja_desde` dela vira agora, então ela vai pro fim da fila. Se não tem outro fiel,
  continua líder. Ser preso depois pelo `!prender` ou pelo `!denunciar` não tira a liderança.
