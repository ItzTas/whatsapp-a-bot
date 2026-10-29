---
name: ia-e-midia
description: "IA e mídia do bot (internal/ai, internal/llm, Asker e cotas, iasemlimite, memória ia_memorias, ferramentas iasorte/iaeconomia, iainventa e iaenxerida/!perseguir, personTexts, !cotacao, !clima, WithMediaJob e binários externos, GIFs e figurinhas embutidos, !buscar). Carregue antes de mexer em comando de IA, de mídia ou de busca."
user-invocable: false
---

# IA e mídia

Camadas da IA, ferramentas dela e os comandos de mídia.

- `llm.Assistant` (`internal/llm/assistente.go`) tenta as camadas em ordem (provedor com cota,
  depois o modelo local); `chatModel` (`internal/llm/llm.go`) abstrai Anthropic SDK e APIs
  compatíveis com OpenAI.
- Toda chamada à IA passa um `llm.Asker` (`b.Asker(ctx, chat, user)`). Com o submodo
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
  profissão (com as vantagens, via `Details()`) de quem tá no grupo, e recusam com o modo
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
- O `!info` (`info.go`) procura na Wikipédia em português: a busca do `rest.php` traz até
  `infoCandidates` páginas, e o resumo (`page/summary`) de cada uma é lido em ordem até achar uma
  que não seja desambiguação. Vai o resumo com a foto (JPEG ou PNG) e o link, mais os outros
  títulos achados. A Wikimedia recusa (403) pedido sem `User-Agent` próprio, por isso o `infoGet`
  manda o `infoUserAgent`. Sem texto depois do comando, procura o texto da mensagem respondida.
  Se o assunto começa com `!` (`!info !imoveis`), não vai pra Wikipédia: o `b.CommandInfo`
  (`internal/core/comandos.go`) acha os comandos com aquele nome ou alias (os de dono só pra
  quem é dono) e mostra a ajuda deles, avisando quando o modo do comando tá desligado na conversa.
- Trabalho pesado com binários externos passa por `b.WithMediaJob`
  (`internal/core/utilidades.go`): limita a 2
  jobs simultâneos, aplica timeout, roda com `nice` e reage com ⏳/✅/❌ na mensagem.
- GIFs (`internal/core/assets/<nome>/*.mp4`) e figurinhas (`internal/core/figurinhas/`) são
  embutidos no binário.
- O `!buscar` (`buscar.go`) lê as páginas de busca do Buscapé e da KaBuM (JSON do
  `__NEXT_DATA__`) e da Amazon (HTML, pulando os patrocinados `AdHolder`) com o `net/http`
  padrão, em paralelo, e tenta de novo uma vez quando a loja responde 503/429. Cada loja vira
  uma mensagem: as fotos dos produtos lado a lado (montadas com `magick montage`, sem texto,
  porque a imagem Docker não tem fontes) e a lista na legenda; se as fotos falharem, vai só o
  texto. OLX, Mercado
  Livre, Magalu, Shopee e Facebook Marketplace foram testados e bloqueiam o bot (Cloudflare,
  antibot ou login), então não adianta tentar raspar eles direto.
