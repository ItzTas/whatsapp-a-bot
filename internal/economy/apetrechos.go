package economy

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
}
