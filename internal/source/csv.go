// Package source downloads and parses the Statistik Austria street list.
package source

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"asv6/internal/index"
)

// DefaultURL is the official Statistik Austria street directory.
const DefaultURL = "https://www.statistik.at/verzeichnis/strassenliste/gemplzstr.csv"

// Fetch downloads and parses the CSV at url.
func Fetch(ctx context.Context, url string) ([]index.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return Parse(resp.Body)
}

// Parse reads the CSV: lines 1-2 are skipped, line 3 is the header.
func Parse(r io.Reader) ([]index.Entry, error) {
	br := bufio.NewReader(r)
	for range 2 {
		if _, err := br.ReadString('\n'); err != nil {
			return nil, fmt.Errorf("skip preamble: %w", err)
		}
	}
	cr := csv.NewReader(br)
	cr.Comma = ';'
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.ReuseRecord = true

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimSpace(strings.TrimPrefix(h, "\uFEFF"))] = i
	}
	get := func(rec []string, name string) string {
		if i, ok := col[name]; ok && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	if _, ok := col["Postleitzahl"]; !ok {
		return nil, errors.New("missing column Postleitzahl")
	}

	var out []index.Entry
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		plz, ok := postalCode(get(rec, "Postleitzahl"))
		if !ok {
			continue // footer / malformed line
		}
		for _, m := range distinct(get(rec, "Politische Gemeinde"), get(rec, "Zustellort")) {
			for _, s := range distinct(get(rec, "Straßenname"), get(rec, "Straßennamen Kurz")) {
				out = append(out, index.Entry{PostalCode: plz, Municipality: m, Street: s})
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no entries parsed")
	}
	return out, nil
}

// WithoutSonderadressen drops placeholder entries like "Sonderadresse 1.Bezirk".
func WithoutSonderadressen(entries []index.Entry) []index.Entry {
	return slices.DeleteFunc(entries, func(e index.Entry) bool {
		return strings.Contains(strings.ToLower(e.Street), "sonderadresse")
	})
}

// postalCode left-pads numeric codes to 4 digits.
func postalCode(s string) (string, bool) {
	if s == "" || len(s) > 4 {
		return "", false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	return strings.Repeat("0", 4-len(s)) + s, true
}

func distinct(a, b string) []string {
	switch {
	case a == "":
		if b == "" {
			return nil
		}
		return []string{b}
	case b == "" || a == b:
		return []string{a}
	}
	return []string{a, b}
}
