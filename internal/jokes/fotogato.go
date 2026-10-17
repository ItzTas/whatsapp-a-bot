package jokes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/ItzTas/personal-projects/whatsapp-a-bot/internal/core"
	"go.mau.fi/whatsmeow/types/events"
)

const (
	catAPIEndpoint   = "https://api.thecatapi.com/v1/images/search?mime_types=jpg,png"
	catTimeout       = 20 * time.Second
	catImageMaxBytes = 10 << 20
	catDown          = "😿 Os gatos fugiram, a API não respondeu. Tenta de novo daqui a pouco"
)

var catCaptions = []string{
	"🐱 Miau",
	"😺 Olha esse bichano",
	"🐈 Gato do dia entregue",
	"😻 Não tem como não amar",
	"🐾 Pediu gato, chegou gato",
}

func (b *bot) fotogato(ctx context.Context, evt *events.Message) error {
	image, err := fetchCatPhoto(ctx)
	if err != nil {
		b.Client.Log.Warnf("!fotogato: %v", err)
		return b.ReplyTo(ctx, evt, catDown)
	}
	return b.SendImage(ctx, evt.Info.Chat.ToNonAD(), core.Outgoing{
		Data:     image,
		Mimetype: http.DetectContentType(image),
		Caption:  catCaptions[rand.IntN(len(catCaptions))],
	}, core.ReplyTo(evt))
}

func fetchCatPhoto(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, catTimeout)
	defer cancel()
	link, err := fetchCatLink(ctx)
	if err != nil {
		return nil, fmt.Errorf("buscar link do gato: %w", err)
	}
	image, err := core.HTTPGet(ctx, link, catImageMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("baixar foto %s: %w", link, err)
	}
	return image, nil
}

func fetchCatLink(ctx context.Context) (string, error) {
	body, err := core.HTTPGet(ctx, catAPIEndpoint, 1<<20)
	if err != nil {
		return "", err
	}
	var cats []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &cats); err != nil {
		return "", err
	}
	if len(cats) == 0 || cats[0].URL == "" {
		return "", errors.New("API respondeu sem nenhum gato")
	}
	return cats[0].URL, nil
}
