// Package index implements an immutable in-memory trigram index over addresses.
package index

import (
	"slices"
	"sync"
	"time"

	"asv6/internal/normalize"
)

// Entry is one address source record.
type Entry struct {
	PostalCode   string
	Municipality string
	Street       string
}

// Index is immutable after Build and safe for concurrent searches.
type Index struct {
	docs     []string            // display form: "PLZ Municipality, Street"
	docLen   []uint16            // unique trigram count per doc
	postings map[uint32][]uint32 // trigram -> sorted doc ids
	pool     sync.Pool
	BuiltAt  time.Time
}

// Len returns the number of indexed addresses.
func (ix *Index) Len() int { return len(ix.docs) }

// Build creates an index, deduplicating on the canonical address form.
func Build(entries []Entry) *Index {
	ix := &Index{postings: make(map[uint32][]uint32, 1<<16), BuiltAt: time.Now()}
	seen := make(map[string]struct{}, len(entries))
	var buf []uint32
	for _, e := range entries {
		display := e.PostalCode + " " + e.Municipality + ", " + e.Street
		canon := normalize.Canonical(display)
		if canon == "" {
			continue
		}
		if _, dup := seen[canon]; dup {
			continue
		}
		seen[canon] = struct{}{}

		buf = trigrams(buf[:0], pad(canon))
		slices.Sort(buf)
		buf = slices.Compact(buf)

		id := uint32(len(ix.docs))
		ix.docs = append(ix.docs, display)
		ix.docLen = append(ix.docLen, uint16(min(len(buf), 0xFFFF)))
		for _, k := range buf {
			ix.postings[k] = append(ix.postings[k], id)
		}
	}
	ix.compact()
	n := len(ix.docs)
	ix.pool.New = func() any { return newScratch(n) }
	return ix
}

// compact moves all posting lists into one contiguous arena to drop append slack.
func (ix *Index) compact() {
	total := 0
	for _, p := range ix.postings {
		total += len(p)
	}
	arena := make([]uint32, 0, total)
	for k, p := range ix.postings {
		s := len(arena)
		arena = append(arena, p...)
		ix.postings[k] = arena[s:len(arena):len(arena)]
	}
}
