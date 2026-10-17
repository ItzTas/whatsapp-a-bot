package utilities

import (
	"fmt"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
)

func (b *bot) utilitiesCommands() []core.Command {
	return []core.Command{
		{
			Name:     "baixar",
			Aliases:  []string{"download", "dl"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!baixar <link>", "baixa o vídeo, a imagem ou o gif do link e manda aqui 📥 (com o modo baixarlinks ligado, os links do Instagram, TikTok, X e outros nem precisam do comando)"},
				{"!baixar", "responda uma mensagem que tenha link e o bot baixa o que tem nele, ou responda uma figurinha e o bot manda ela como imagem (ou gif, se for animada)"},
				{"!baixar @pessoa", "baixa a foto de perfil da pessoa marcada e manda aqui 🖼️"},
			},
			Run: b.baixar,
		},
		{
			Name:     "mp3",
			Aliases:  []string{"audio"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!mp3 <link>", "baixa só o áudio do link e manda como arquivo mp3, com o nome e a capa 🎵"},
				{"!mp3", "responda um vídeo, um áudio ou uma mensagem com link e o bot manda o áudio em mp3"},
			},
			Run: b.mp3,
		},
		{
			Name:     "musica",
			Aliases:  []string{"tocar", "play"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!musica <nome>", "procura a música no YouTube e manda o áudio aqui 🎶 Dica: põe o artista junto, tipo !musica entombed deftones"},
				{"!musica <link>", "manda o áudio do link aqui pra tocar direto 🎧"},
				{"!musica meme <nome>", "manda um áudio de meme do MyInstants, igual o !som"},
			},
			Run: b.musica,
		},
		{
			Name:     "som",
			Aliases:  []string{"sons"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!som <nome>", "procura o áudio de meme no MyInstants e manda aqui 🔊 tipo !som vine boom"},
			},
			Run: b.som,
		},
		{
			Name:     "transcrever",
			Aliases:  []string{"transcricao"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!transcrever", "responda um áudio ou um vídeo com !transcrever e o bot escreve o que foi falado 🎙️"},
			},
			Run: b.transcrever,
		},
		{
			Name:     "figurinha",
			Aliases:  []string{"fig", "s", "sticker"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!figurinha", "mande uma foto, vídeo ou gif com !figurinha na legenda, ou responda um deles, e o bot transforma em figurinha 🖼️"},
			},
			Run: b.figurinha,
		},
		{
			Name:     "lembrete",
			Aliases:  []string{"lembretes", "lembrar"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!lembrete <hora> <texto>", fmt.Sprintf("cria um lembrete que o bot manda nesta conversa todo dia nessa hora, tipo !lembrete 8h30 tomar água ⏰ Dá pra marcar gente no texto, e cada conversa pode ter até %d", reminderMaxPerChat)},
				{"!lembrete", "mostra os lembretes da conversa, com a hora e quem criou cada um"},
				{"!lembrete apagar <número>", "apaga o lembrete com esse número na lista (só quem criou ou um admin do grupo)"},
			},
			Run: b.lembrete,
		},
		{
			Name:     "buscar",
			Aliases:  []string{"procurar"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!buscar <produto>", fmt.Sprintf("procura o produto no Buscapé, na Amazon e na KaBuM e manda os %d primeiros de cada um, com preço e link 🛒", shoppingPerShop)},
			},
			Run: b.buscar,
		},
		{
			Name:     "aniversario",
			Aliases:  []string{"niver", "aniversarios"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!aniversario <dd/mm>", "salva o seu aniversário nesta conversa, e o bot manda os parabéns aqui assim que der meia-noite do dia 🎂"},
				{"!aniversario @pessoa <dd/mm>", "salva o aniversário de outra pessoa"},
				{"!aniversario [@pessoa] hoje", "salva o aniversário como hoje, e os parabéns saem no mesmo dia"},
				{"!aniversario", "mostra os próximos aniversários da conversa"},
				{"!aniversario apagar [@pessoa]", "apaga o seu aniversário, ou o de outra pessoa (só admin do grupo)"},
			},
			Run: b.aniversario,
		},
		{
			Name:     "cotacao",
			Aliases:  []string{"dolar", "cambio"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!cotacao", "mostra quanto tão o dólar, o euro, a libra, o bitcoin e o ethereum em reais, com a variação do dia 💱"},
				{"!cotacao <moeda>", "mostra a cotação de uma moeda pelo código ou pelo nome, tipo !cotacao ARS ou !cotacao iene"},
				{"!cotacao [moeda] em dolar", "mesma coisa, só que em dólar em vez de real, tipo !cotacao em dolar ou !cotacao bitcoin em dolar"},
			},
			Run: b.cotacao,
		},
		{
			Name:     "clima",
			Aliases:  []string{"tempo", "previsao"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!clima <cidade>", "mostra como tá o tempo agora na cidade e a previsão dos próximos dias 🌦️"},
			},
			Run: b.clima,
		},
		{
			Name:     "ocr",
			Aliases:  []string{"ler", "texto"},
			Category: core.UtilitiesCategory,
			Help: []core.Usage{
				{"!ocr", "mande uma foto com !ocr na legenda, ou responda uma foto ou figurinha, e o bot escreve o texto que tiver nela 🔍"},
			},
			Run: b.ocr,
		},
	}
}
