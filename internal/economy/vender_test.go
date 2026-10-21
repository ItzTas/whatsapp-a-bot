package economy

import "testing"

func TestIsHostageSale(t *testing.T) {
	cases := map[string]bool{
		"!vender refem":       true,
		"!vender refém":       true,
		"!vender Refém":       true,
		"!vender sequestrado": true,
		"!vender refem 2":     false,
		"!vender peixe":       false,
		"!vender":             false,
	}
	for text, want := range cases {
		if got := isHostageSale(text); got != want {
			t.Errorf("isHostageSale(%q) = %v, quero %v", text, got, want)
		}
	}
}
