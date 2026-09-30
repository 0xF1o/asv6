package normalize

import "testing"

func TestCanonical(t *testing.T) {
	cases := map[string]string{
		"1110 Wien, Gasometer": "1110 wien gasometer",
		"Straße Ärger Öl Übel": "strasse aerger oel uebel",
		"St.-Antoni-Str.":      "st antoni str",
		"  A/1   3.Stock 22b ": "a 1 3 stock 22b",
		"Hôtel-de-Ville":       "hotel de ville",
	}
	for in, want := range cases {
		if got := Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}
