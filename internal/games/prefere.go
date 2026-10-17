package games

import (
	"context"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types/events"
)

var prefereOpcoes = [][2]string{
	{"Beijar seu ex", "Beijar o ex do seu melhor amigo(a)"},
	{"Transar só no escuro pra sempre", "Transar só com a luz acesa pra sempre"},
	{"Seus pais lerem todas as suas conversas", "Seu ex ver toda a sua galeria"},
	{"Ficar 1 ano sem sexo", "Ficar 1 ano sem celular"},
	{"Ser pego(a) transando pelos seus pais", "Pegar seus pais transando"},
	{"Um ménage", "Uma noite inteira só com o seu crush"},
	{"Ficar com alguém deste grupo", "Nunca mais beijar ninguém"},
	{"Seu histórico de pesquisa vazar", "Suas mensagens apagadas vazarem"},
	{"Sexo selvagem de 5 minutos", "Sexo romântico de 2 horas"},
	{"Mandar nude pro grupo da família", "Mandar áudio gemendo pro grupo do trabalho"},
	{"Namorar alguém que beija mal", "Namorar alguém que é ruim de cama"},
	{"Transar na praia", "Transar no elevador"},
	{"Dominar", "Ser dominado(a)"},
	{"Seu crush descobrir que você gosta dele(a)", "Seu ex descobrir que você ainda pensa nele(a)"},
	{"Gemer alto sem querer na casa da sogra", "Ouvir a sogra gemendo"},
	{"Ficar com o seu chefe", "Ficar com o seu ex-professor(a)"},
	{"Uma noite com uma celebridade e ninguém acreditar", "Uma noite com alguém normal e todo mundo saber"},
	{"Só poder beijar, nunca transar", "Só poder transar, nunca beijar"},
	{"Seu ex ser o melhor sexo da sua vida", "Seu atual ser o pior"},
	{"Ser flagrado(a) no motel pelo seu pai", "Encontrar o seu pai no motel"},
	{"Ler todas as conversas do seu crush", "Seu crush ler todas as suas"},
	{"Ser chifrado(a) e saber", "Ser chifrado(a) e nunca descobrir"},
	{"Namorar alguém deste grupo", "Ficar solteiro(a) pra sempre"},
	{"Perdoar uma traição", "Terminar e nunca mais falar com a pessoa"},
	{"Seu parceiro(a) ter um melhor amigo(a) que é ex", "Ter que conviver com a família dele(a) todo fim de semana"},
	{"Ser rico(a) e feio(a)", "Ser pobre e gostoso(a)"},
	{"Todo mundo do grupo ver suas pesquisas", "Todo mundo do grupo ouvir seus áudios apagados"},
	{"Peidar alto num primeiro encontro", "Arrotar no meio do beijo"},
	{"Nunca mais tomar banho quente", "Nunca mais usar o celular no banheiro"},
	{"Ter chulé pra sempre", "Ter bafo pra sempre"},
	{"Comer um prato de comida do chão", "Lamber o celular de alguém daqui"},
	{"Falar tudo o que pensa por 1 dia", "Ficar mudo(a) por 1 semana"},
	{"Tomar um fora em público", "Ser pedido(a) em casamento em público e dizer não"},
	{"Ouvir só funk pra sempre", "Ouvir só sertanejo pra sempre"},
	{"Ter seu nome no grupo trocado pra Corno(a) por 1 mês", "Mudar sua foto de perfil pra uma do Faustão por 1 mês"},
	{"Voltar com o seu ex", "Ficar com o ex do seu melhor amigo(a)"},
	{"Seu pai ler seu histórico de pesquisa", "Sua mãe ler suas conversas"},
	{"Dormir de conchinha com alguém deste grupo", "Dormir sozinho(a) num quarto com uma barata solta"},
}

func (b *bot) prefere(ctx context.Context, evt *events.Message) error {
	opcoes := prefereOpcoes[rand.IntN(len(prefereOpcoes))]
	return b.SendPoll(ctx, evt.Info.Chat, "🤔 Você prefere...", opcoes[:])
}
