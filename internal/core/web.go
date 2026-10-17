package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

type Outgoing struct {
	Data      []byte
	File      string
	Name      string
	Mimetype  string
	Caption   string
	Thumbnail []byte
	Video     bool
	GIF       bool
	Document  bool
}

func Attr(n *html.Node, key string) string {
	if n.Type != html.ElementNode {
		return ""
	}
	i := slices.IndexFunc(n.Attr, func(a html.Attribute) bool { return a.Key == key })
	if i < 0 {
		return ""
	}
	return n.Attr[i].Val
}

func HasClass(n *html.Node, class string) bool {
	return slices.Contains(strings.Fields(Attr(n, "class")), class)
}

func TextContent(root *html.Node) string {
	var sb strings.Builder
	for n := range root.Descendants() {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
	}
	return sb.String()
}

func HTTPGet(ctx context.Context, link string, maxBytes int64) (data []byte, err error) {
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
