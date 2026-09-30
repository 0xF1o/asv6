package source

import (
	"os"
	"strings"
	"testing"
	"time"

	"asv6/internal/index"
)

const sample = `Gemeinde-, Ortschafts-, Straßen- und PLZ-Verzeichnis;;;;;;;;;
Erstellt am:;24.09.2026 08:31:28;;;;;;;;
Gemeindekennziffer;Politische Gemeinde;Ortschaftskennziffer;Ortschaftsname;Straßenkennziffer;Straßenname;Straßennamen Kurz;Postleitzahl;Gemeindecode;Zustellort
10101;Eisenstadt;00001;Eisenstadt;000010;Sankt-Antoni-Straße;St.-Antoni-Str.;7000;10101;Eisenstadt
10719;Sankt Andrä am Zicksee;00165;Sankt Andrä am Zicksee;130963;Mobilheimplatz C I;Mobilheimplatz C I;7161;10719;St.Andrä/Zicksee
10101;X;1;X;1;Teststraße;;100;1;X
Quelle: STATISTIK AUSTRIA. erstellt am 24.09.2026 08:31:28
`

func TestParse(t *testing.T) {
	got, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	want := []index.Entry{
		{PostalCode: "7000", Municipality: "Eisenstadt", Street: "Sankt-Antoni-Straße"},
		{PostalCode: "7000", Municipality: "Eisenstadt", Street: "St.-Antoni-Str."},
		{PostalCode: "7161", Municipality: "Sankt Andrä am Zicksee", Street: "Mobilheimplatz C I"},
		{PostalCode: "7161", Municipality: "St.Andrä/Zicksee", Street: "Mobilheimplatz C I"},
		{PostalCode: "0100", Municipality: "X", Street: "Teststraße"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d entries: %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestWithoutSonderadressen(t *testing.T) {
	got := WithoutSonderadressen([]index.Entry{
		{PostalCode: "1010", Municipality: "Wien", Street: "Sonderadresse 1.Bezirk"},
		{PostalCode: "1010", Municipality: "Wien", Street: "Stephansplatz"},
	})
	if len(got) != 1 || got[0].Street != "Stephansplatz" {
		t.Fatalf("got %+v", got)
	}
}

// Set ASV_CSV to a local copy of gemplzstr.csv to benchmark against real data.
func BenchmarkSearchRealData(b *testing.B) {
	path := os.Getenv("ASV_CSV")
	if path == "" {
		b.Skip("ASV_CSV not set")
	}
	f, err := os.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	t0 := time.Now()
	entries, err := Parse(f)
	if err != nil {
		b.Fatal(err)
	}
	ix := index.Build(entries)
	b.Logf("rows=%d docs=%d build=%s", len(entries), ix.Len(), time.Since(t0))

	queries := []string{
		"1110 Wien, Gasometer A/1 3.Stock 22b",
		"Wien Hauptstrasse 5",
		"8010 Graz Herrengasse 16",
		"mariahilfer str 120/3/12",
		"St. Andrä Zicksee Mobilheimplatz",
		"a",
	}
	var worst time.Duration
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q := queries[i%len(queries)]
		s := time.Now()
		ix.Search(q, 10, 0.15)
		if d := time.Since(s); d > worst {
			worst = d
		}
	}
	b.ReportMetric(float64(worst.Microseconds()), "worst-µs")
}
