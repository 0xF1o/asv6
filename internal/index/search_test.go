package index

import "testing"

func testIndex() *Index {
	return Build([]Entry{
		{"1110", "Wien", "Gasometer"},
		{"1110", "Wien", "Gasometergasse"},
		{"1110", "Wien", "Simmeringer Hauptstraße"},
		{"1110", "Wien", "Simmeringer Hauptstraße"},  // duplicate
		{"1110", "wien", "Simmeringer Hauptstrasse"}, // canonical duplicate
		{"7161", "Sankt Andrä am Zicksee", "Mobilheimplatz C I"},
		{"7161", "Sankt Andrä am Zicksee", "Mobilheimplatz C"},
	})
}

func TestDedup(t *testing.T) {
	if n := testIndex().Len(); n != 5 {
		t.Fatalf("Len = %d, want 5", n)
	}
}

func TestSearchTrimsStreetNumber(t *testing.T) {
	res := testIndex().Search("1110 Wien, Gasometer A/1 3.Stock 22b", 10, 0.15)
	if len(res) == 0 {
		t.Fatal("no results")
	}
	top := res[0]
	if top.Address != "1110 Wien, Gasometer" || top.StreetNumber != "A/1 3.Stock 22b" || top.Score != 1 {
		t.Fatalf("top = %+v", top)
	}
}

func TestSearchFullMatchKeepsWords(t *testing.T) {
	res := testIndex().Search("Mobilheimplatz C I", 10, 0.15)
	if res[0].Address != "7161 Sankt Andrä am Zicksee, Mobilheimplatz C I" || res[0].StreetNumber != "" {
		t.Fatalf("top = %+v", res[0])
	}
	if res[1].Address != "7161 Sankt Andrä am Zicksee, Mobilheimplatz C" || res[1].StreetNumber != "" || res[1].Score >= 1 {
		t.Fatalf("second = %+v", res[1])
	}
}

func TestSearchFuzzyUmlaut(t *testing.T) {
	res := testIndex().Search("simmeringer hauptstrase 12", 3, 0.15)
	if len(res) == 0 || res[0].Address != "1110 Wien, Simmeringer Hauptstraße" || res[0].StreetNumber != "12" {
		t.Fatalf("res = %+v", res)
	}
	if res[0].Score <= 0.5 || res[0].Score >= 1 {
		t.Fatalf("unexpected score %v", res[0].Score)
	}
}

func TestMinScoreAndLimit(t *testing.T) {
	ix := testIndex()
	if res := ix.Search("gasometer", 1, 0.15); len(res) != 1 {
		t.Fatalf("limit ignored: %d", len(res))
	}
	for _, r := range ix.Search("xyz", 10, 0.9) {
		if r.Score < 0.9 {
			t.Fatalf("min_score ignored: %+v", r)
		}
	}
	if res := ix.Search("  ,. ", 10, 0); len(res) != 0 {
		t.Fatalf("empty query returned %v", res)
	}
}
