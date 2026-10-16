package main

import (
	"context"
	"fmt"
	"math/rand/v2"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func (b *bot) pedirpix(ctx context.Context, evt *events.Message) error {
	pair, err := b.pixPair(ctx, evt)
	if err != nil {
		return fmt.Errorf("listar participantes: %w", err)
	}
	if len(pair) < 2 {
		return b.sendText(ctx, evt.Info.Chat, "Marca uma ou duas pessoas: !pedirpix @pessoa ou !pedirpix @pessoa1 @pessoa2")
	}

	valores := []string{
		"R$ 0,50", "R$ 2,37", "R$ 5", "R$ 12,90", "R$ 20 (é só 20, pô)",
		"R$ 50", "R$ 150", "R$ 1.000", "R$ 9.999,99", "R$ 50 mil",
	}
	motivos := []string{
		"pra comprar um salgado na cantina 🥟",
		"pra pagar o Uber que já tá chegando 🚗",
		"pra inteirar o rodízio 🍣",
		"porque esqueceu o cartão em casa (de novo) 💳",
		"pra comprar figurinha da Copa ⚽",
		"pra pagar a conta de luz atrasada desde março 💡",
		"porque faz aniversário... daqui a 8 meses 🎂",
		"pra comprar ração pro gato que nem existe 🐈",
		"pra jogar no tigrinho 🐯",
		"pra pagar o boleto do curso de coach 📈",
		"pra devolver amanhã (spoiler: não vai) 🤞",
		"pra comprar um Danone 🥛",
	}
	chaves := []string{
		"o CPF da mãe",
		"o número da vó",
		"uma chave aleatória que nem o dono sabe de cor",
		"o e-mail do Orkut",
		"o CNPJ do MEI de brigadeiro",
		"o celular de 2014 que nem liga mais",
	}
	desfechos := []string{
		"✅ Mandou na hora. Que amizade 🥹",
		"👀 Visualizou e não respondeu",
		"🪙 Mandou R$ 0,01 só de raiva",
		"📵 Fingiu que tava sem sinal",
		"🧾 Mandou o comprovante de um pix de 2019",
		"😂 Respondeu \"kkkkkkk\" e sumiu",
		"🫠 Disse que também tá liso",
		"🚫 Bloqueou na hora",
		"💸 Mandou o dobro e pediu pra não contar pra ninguém",
		"🤝 Topou, mas com juros de 30% ao dia",
	}

	from, to := pair[0], pair[1]
	text := fmt.Sprintf("💸 @%s pediu um pix de %s pra @%s\n\n📝 Motivo: %s\n🔑 Chave: %s\n🎯 Chance de receber: %d%%\n\n%s",
		from.User,
		valores[rand.IntN(len(valores))],
		to.User,
		motivos[rand.IntN(len(motivos))],
		chaves[rand.IntN(len(chaves))],
		rand.IntN(101),
		desfechos[rand.IntN(len(desfechos))],
	)
	_, err = b.sendMentions(ctx, evt.Info.Chat, text, []types.JID{from, to})
	return err
}

func (b *bot) pixPair(ctx context.Context, evt *events.Message) ([]types.JID, error) {
	mentions := mentionedJIDs(evt.Message)
	if len(mentions) == 0 {
		return b.randomPartner(ctx, evt)
	}
	pair, ok := coupleFrom(evt.Info.Sender, mentions)
	if !ok {
		return nil, nil
	}
	return pair[:], nil
}
