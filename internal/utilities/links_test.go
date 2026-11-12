package utilities

import "testing"

func TestFindLink(t *testing.T) {
	cases := map[string]string{
		"!baixar olha isso https://x.com/a/status/1 kkkk":           "https://x.com/a/status/1",
		"!baixar vê aí: https://youtu.be/abc?si=x=, muito bom":      "https://youtu.be/abc?si=x=",
		"!baixar (https://pt.wikipedia.org/wiki/Gato_(animal)) top": "https://pt.wikipedia.org/wiki/Gato_(animal)",
		"!baixar esse https://www.instagram.com/reel/xyz/🔥🔥 aqui":   "https://www.instagram.com/reel/xyz/",
		"!baixar texto https://tiktok.com/@a/video/9 texto":         "https://tiktok.com/@a/video/9",
		"!baixar manda www.reddit.com/r/gatos/comments/1 pfv":       "https://www.reddit.com/r/gatos/comments/1",
		"!baixar HTTPS://Imgur.com/Abc.mp4!!!":                      "HTTPS://Imgur.com/Abc.mp4",
		"!baixar sem link nenhum aqui":                              "",
	}
	for text, want := range cases {
		if got := findLink(text); got != want {
			t.Errorf("findLink(%q) = %q, quero %q", text, got, want)
		}
	}
}
