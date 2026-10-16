package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"time"

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
		b.client.Log.Warnf("!fotogato: %v", err)
		return b.replyTo(ctx, evt, catDown)
	}
	return b.sendImage(ctx, evt.Info.Chat.ToNonAD(), outgoing{
		data:     image,
		mimetype: http.DetectContentType(image),
		caption:  catCaptions[rand.IntN(len(catCaptions))],
	}, replyTo(evt))
}

func fetchCatPhoto(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, catTimeout)
	defer cancel()
	link, err := fetchCatLink(ctx)
	if err != nil {
		return nil, fmt.Errorf("buscar link do gato: %w", err)
	}
	image, err := httpGet(ctx, link, catImageMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("baixar foto %s: %w", link, err)
	}
	return image, nil
}

func fetchCatLink(ctx context.Context) (string, error) {
	body, err := httpGet(ctx, catAPIEndpoint, 1<<20)
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

func httpGet(ctx context.Context, link string, maxBytes int64) (data []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %s", resp.Status)
	}
	data, err = io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("resposta passou de %d bytes", maxBytes)
	}
	return data, nil
}
