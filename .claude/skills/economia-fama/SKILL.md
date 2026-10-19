---
name: economia-fama
description: "Fama da economia (fama.go, !postar nas redes de platforms, tabela fama com os seguidores por rede, postagem_em, queda de seguidor faded, níveis fameTiers e celebrity(), alcance reach/stage da profissão, !fama, !publi com cachê e teto por nível, as desvantagens de ser famoso no !venderrefem, no !fugir e no !denunciar). Carregue antes de mexer em seguidor, rede, nível de fama ou publi."
user-invocable: false
---

# Economia: fama

- O `!postar <rede>` (`fama.go`) grava os seguidores por rede na tabela `fama` (`chat`, `pessoa`,
  `plataforma`, `seguidores`, `postado_em`), e o total é a soma das redes. As redes vivem em código
  (`platforms`) e só o `key` vai pra coluna `plataforma`: nunca renomeie um `key`, senão os
  seguidores daquela rede somem. A espera é uma só pra todas as redes (`postCooldown`, coluna
  `postagem_em` da `carteiras`), então postar numa rede é deixar de postar nas outras. O
  `postEvery` monta a espera: aplica o decreto `fama` do presidente (`fameDecree`, de
  -`decreeSwing` a +`decreeSwing` por cento) e depois o `hasted`. O `!postar` e o `!fama` passam
  por ele.
- Postar pede um aparelho na mochila: os itens com `networks` (as `key` das redes que eles
  liberam), hoje o 📱 Celular (TikTok e Instagram) e o 💻 Computador (YouTube e Twitch), os dois
  na `!loja`. O `!postar` lê os aparelhos (`walletStore.devices`, pelo `holding`) em paralelo com
  o `ProfessionOf` e recusa com o `deviceNames()` da rede quando nenhum aparelho dela tá na
  mochila (`postableFrom`). Vale só de ter, não gasta, e a quantidade não importa. O `!fama` avisa
  em cada rede sem aparelho e o `postUsage` mostra o aparelho de cada rede. Rede nova precisa
  entrar no `networks` de algum aparelho, senão ninguém consegue postar nela.
- Cada post (`platform.roll`) sorteia, nesta ordem: cancelamento (`cancel`, perde de `loss[0]` a
  `loss[1]` por cento dos seguidores daquela rede, só com seguidor), flop (`flop`, não ganha
  nada) ou post normal, que ganha `base` mais uma fatia dos seguidores atuais (`growth`) e pode
  viralizar (`viral`: soma um valor log-uniforme entre `viralMin` e `viralMax` e outra fatia,
  `viralGrowth`). As duas fatias encolhem com o tamanho da conta (`room`, que vai a zero no
  `ceiling` da rede, em escala de log), e é isso que deixa ficar famoso muito difícil. O ganho
  todo é multiplicado pelo alcance da profissão. Os números foram calibrados numa simulação: um
  youtuber postando só no YouTube chega a 1 milhão em uns 3 a 5 meses de jogo pesado, e
  profissão sem alcance quase nunca passa de 10 mil. Quem mexer nos números precisa simular de
  novo.
- O alcance (`reachOn`) é uma porcentagem: o campo `reach` da profissão, ou `artReach` pra área
  de Arte e mídia, ou `baseReach` pro resto. A rede da profissão (`stage`, o `key` de uma rede)
  rende `stageBonus` por cento a mais. Profissão nova de mídia ganha `reach` e `stage` em
  `profissoes.go`, e o texto sai sozinho no `famePerks`.
- O Chapéu do Boto (relíquia passiva, `flex`) soma `botoReach` no alcance (`p.reach =
  p.reachPercent() + botoReach`, então vale por cima do `reach` da profissão e do `artReach`) e
  liga o `charming`: o `cancelOdds` zera a faixa de cancelamento no `platform.roll` (a chance vai
  pro post normal, não pro flop) e o `backlashOdds` zera o backlash do `!publi`. Por isso o
  `roll` e o `walletStore.post` recebem a profissão inteira, não só o alcance. Na simulação com
  15 posts por dia no YouTube, o youtuber chega a 1 milhão em uns 69 dias sem ele e 43 com ele, e
  quem não tem alcance de profissão sai de quase nunca passar de 10 mil pra 1 milhão em uns 160
  dias.
- A rede parada perde seguidor sem nada rodando em segundo plano: o `faded` calcula na leitura,
  e depois de `grace` sem postar naquela rede, os seguidores caem `decay` por dia (a Twitch tem
  `grace` menor e `decay` maior). O `!postar` grava o valor já com a queda e zera o relógio da
  rede. Por isso tudo que lê seguidores passa pelo `walletStore.audience`, nunca pela coluna
  direto.
- O `ProfessionOf` lê o `audience` em paralelo e aplica o `famed`, então a `Profession` carrega
  as redes em `fans`, e o nível sai do total pelo `celebrity()` (`fameTiers`, do anônimo à
  celebridade). Cada nível acima do anônimo (`famous()`) libera o `!publi` e cobra o preço da
  fama, que cresce por nível:
  - `ransom`: soma no preço do refém no `ransomOf` (`!sequestrar` e `!venderrefem`), junto com o
    `ransomBonus` de quem vende;
  - `escape`: sai da chance do `escapeOdds` (`!fugir` da cadeia, `!fugir @pessoa` e a fuga do
    `!assalto`), porque todo mundo reconhece;
  - `snitch`: soma na chance do `!denunciar` contra a pessoa (`reportOddsOn`).
- O `!publi` paga o `fee` do nível mais uma parte dos seguidores de cada rede (`sponsor` moedas
  por mil seguidores, maior no Instagram), com uma variação de 20% pra cima ou pra baixo, e
  nunca passa do `ceiling` do nível (o teto existe pra fama não virar torneira infinita de
  dinheiro). As moedas são novas e entram pelo `walletStore.claim` (espera `sponsorCooldown`,
  coluna `publi_em`, dízimo e imposto de renda de decreto). Tem `sponsorBacklash` de chance de a
  galera achar forçado e a pessoa perder de `backlashMin` a `backlashMax` por cento dos
  seguidores de todas as redes, cortados direto na tabela (a queda é proporcional, então dá na
  mesma cortar antes ou depois do `faded`).
- O `!fama` mostra as redes com o alcance de cada uma, quanto falta pra começar a cair, o nível,
  o próximo nível, as esperas do `!postar` e do `!publi` e o teto do cachê. O `!tu` mostra uma
  linha com o nível (`fameLine`), e o `!profissao ver` mostra o alcance e o preço da fama
  (`famePerks`).
- As duas colunas de espera (`postagem_em` e `publi_em`) estão no `resetCooldowns`. Os
  seguidores não são espera e o reset não mexe neles.
