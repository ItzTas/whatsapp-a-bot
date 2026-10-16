package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.mau.fi/whatsmeow/types/events"
)

const (
	audioQueryMax        = 100
	songSearchCandidates = 5
	myInstantsURL        = "https://www.myinstants.com"
)

var memeSoundPattern = regexp.MustCompile(`play\('(/media/sounds/[^']+)'`)

type audioSearch struct {
	usage    string
	notFound string
	source   func(query string) audioSource
}

func (b *bot) musica(ctx context.Context, evt *events.Message) error {
	query := textAfterFields(messageText(evt.Message), 1)
	if link := linkPattern.FindString(query); link != "" {
		return b.sendLinkAudio(ctx, evt, link)
	}
	if first, rest, _ := strings.Cut(query, " "); strings.EqualFold(first, "meme") {
		return b.searchAndSendAudio(ctx, evt, strings.TrimSpace(rest), memeSoundSearch)
	}
	return b.searchAndSendAudio(ctx, evt, query, audioSearch{
		usage:    "🎵 Use assim: !musica <nome da música>, tipo !musica entombed deftones. Eu procuro no YouTube e mando o áudio. Também dá pra mandar o link, tipo !musica https://... Pra áudio de meme, usa !musica meme <nome> ou !som <nome>",
		notFound: fmt.Sprintf("Não achei essa música 😕 Tenta escrever o nome junto com o artista. Só mando músicas de até %d minutos", mediaMaxDuration/60),
		source:   songAudio,
	})
}

func (b *bot) sendLinkAudio(ctx context.Context, evt *events.Message, link string) error {
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error {
		return b.extractAndSendAudio(ctx, evt, b.linkAudio(link), b.sendAudioReply)
	})
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("tirar áudio do link %q: %v", link, err)
	return b.replyTo(ctx, evt, audioExtractionFailure)
}

func (b *bot) som(ctx context.Context, evt *events.Message) error {
	return b.searchAndSendAudio(ctx, evt, textAfterFields(messageText(evt.Message), 1), memeSoundSearch)
}

var memeSoundSearch = audioSearch{
	usage:    "🔊 Use assim: !som <nome>, tipo !som vine boom. Eu procuro no MyInstants e mando o áudio do meme",
	notFound: "Não achei esse som 😕 Tenta com outras palavras, tipo o nome do meme em inglês",
	source:   memeAudio,
}

func (b *bot) searchAndSendAudio(ctx context.Context, evt *events.Message, query string, search audioSearch) error {
	if query == "" {
		return b.replyTo(ctx, evt, search.usage)
	}
	if utf8.RuneCountInString(query) > audioQueryMax {
		return b.replyTo(ctx, evt, fmt.Sprintf("Nome grande demais, o máximo é %d letras", audioQueryMax))
	}
	err := b.withMediaJob(ctx, evt, func(ctx context.Context) error {
		return b.extractAndSendAudio(ctx, evt, search.source(query), b.sendAudioReply)
	})
	if err == nil {
		return nil
	}
	b.client.Log.Warnf("buscar áudio %q: %v", query, err)
	return b.replyTo(ctx, evt, search.notFound)
}

func songAudio(query string) audioSource {
	return func(ctx context.Context, dir string) (extractedAudio, error) {
		search := fmt.Sprintf("ytsearch%d:%s", songSearchCandidates, query)
		files, err := runYtDlp(ctx, dir, search, mediaLimits,
			"--max-downloads", "1",
			"--extract-audio", "--audio-format", "mp3", "--audio-quality", mp3Quality)
		if len(files) == 0 {
			return extractedAudio{}, errors.Join(err, errNoMedia)
		}
		return extractedAudio{file: files[0]}, nil
	}
}

func memeAudio(query string) audioSource {
	return func(ctx context.Context, dir string) (extractedAudio, error) {
		page, err := fetchShopPage(ctx, myInstantsURL+"/pt/search/?"+url.Values{"name": {query}}.Encode())
		if err != nil {
			return extractedAudio{}, fmt.Errorf("buscar no MyInstants: %w", err)
		}
		match := memeSoundPattern.FindSubmatch(page)
		if match == nil {
			return extractedAudio{}, errNoMedia
		}
		sound, err := fetchShopPage(ctx, myInstantsURL+string(match[1]))
		if err != nil {
			return extractedAudio{}, fmt.Errorf("baixar som do MyInstants: %w", err)
		}
		return convertToMP3(ctx, dir, sound)
	}
}
