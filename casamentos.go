package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/types"
	"golang.org/x/sync/errgroup"
)

func marriageLength(d time.Duration) string {
	if d < 24*time.Hour {
		return formatDuration(d)
	}
	return countLabel(int(d.Hours()/24), "dia", "dias")
}

type weddingNews struct {
	renewal  bool
	marriage relationship
	lines    []string
	mentions []types.JID
}

func (b *bot) weddingNews(ctx context.Context, chat types.JID, couple [2]types.JID) (weddingNews, error) {
	var (
		all     []relationship
		aliases [2][]string
	)
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) { all, err = b.relationships.inChat(gctx, chat, kindMarriage); return err })
	for i, p := range couple {
		g.Go(func() error { aliases[i] = b.userAliases(gctx, p); return nil })
	}
	if err := g.Wait(); err != nil {
		return weddingNews{}, err
	}

	var news weddingNews
	if i := slices.IndexFunc(all, func(m relationship) bool { return m.joins(aliases[0], aliases[1]) }); i >= 0 {
		news.renewal = true
		news.marriage = all[i]
		news.lines = append(news.lines, "💒 Os dois já eram casados, então só renovaram os votos")
	}
	for i, p := range couple {
		partners := otherPartners(all, aliases[i], aliases[1-i])
		if len(partners) == 0 {
			continue
		}
		news.lines = append(news.lines, fmt.Sprintf("⚠️ Bigamia! @%s já era casado(a) com %s", p.User, joinMentions(partners)))
		news.mentions = append(news.mentions, partners...)
	}
	return news, nil
}

func (n weddingNews) caption() string {
	if len(n.lines) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(n.lines, "\n")
}
