package economy

import "fmt"

var gatherGoods = []item{
	{id: "molinete", name: "Molinete", emoji: "⚙️", rarity: rare, value: 1_800, shop: true, aliases: []string{"carretilha"},
		gear: &gear{activities: []string{"!pescar"}, luck: 1}},
	{id: "barco-a-remo", name: "Barco a remo", emoji: "🚣", rarity: rare, value: 2_500, shop: true, aliases: []string{"barco", "canoa"},
		gear: &gear{activities: []string{"!pescar"}, faster: 30}},
	{id: "polvora", name: "Pólvora de garimpo", emoji: "🧨", rarity: uncommon, value: 50, shop: true, aliases: []string{"polvora de garimpo"},
		gear: &gear{activities: []string{"!minerar"}, luck: 1, spent: true, skipsRelics: true}},
	{id: "carrinho-de-mao", name: "Carrinho de mão", emoji: "🛞", rarity: uncommon, value: 700, shop: true, aliases: []string{"carrinho de obra"},
		gear: &gear{activities: []string{"!vasculhar"}, faster: 30}},
	{id: "peneira", name: "Peneira de entulho", emoji: "🧺", rarity: uncommon, value: 900, shop: true, aliases: []string{"peneira de entulho"},
		gear: &gear{activities: []string{"!vasculhar"}, fewerJunk: 40}},
	{id: "guia-de-insetos", name: "Guia de campo de insetos", emoji: "📗", rarity: rare, value: 1_200, shop: true, aliases: []string{"guia de insetos", "guia de campo"},
		gear: &gear{activities: []string{"!catar"}, luck: 1}},
	{id: "mochila-de-trilha", name: "Mochila de trilha", emoji: "🥾", rarity: rare, value: 3_000, shop: true, aliases: []string{"trilha"},
		gear: &gear{activities: []string{"!pescar", "!minerar", "!vasculhar", "!catar"}, fewerJunk: 15}},
	{id: "detector-de-metais", name: "Detector de metais", emoji: "📟", rarity: epic, value: 6_000, shop: true, rotating: true, aliases: []string{"detector"},
		gear: &gear{activities: []string{"!minerar", "!vasculhar"}, moreRare: 40}},
	{id: loadedCoinID, name: "Moeda viciada", emoji: "🪙", rarity: epic, value: 6_500, illegal: true, daily: loadedCoinDaily, aliases: []string{"moeda adulterada", "moeda de duas caras"},
		flex: &flex{
			effect: fmt.Sprintf("deixa os jogos do %s um pouco mais a seu favor e dá +%d%% de lucro quando ganha, mas todo ganho com ela te deixa *alvo fácil* na hora, e cada aposta tem %d%% de chance de te pegarem: a aposta não vale, a moeda é apreendida, o cassino toma de %d%% a %d%% do saldo e você vai preso ou morre",
				casinoGames, loadedCoinLuck, percent(loadedCoinCaught), loadedCoinLossMin, loadedCoinLossMax),
			brief: fmt.Sprintf("mais chance e +%d%% de lucro nas apostas, mas ganhar te deixa alvo fácil e pode dar cadeia ou morte", loadedCoinLuck),
			apply: func(p Profession) Profession {
				p.cheat += loadedCoinFavor
				p.casinoLuck += loadedCoinLuck
				return p
			}}},
}
