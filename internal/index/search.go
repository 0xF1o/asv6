package index

import (
	"math"
	"slices"
	"strings"

	"asv6/internal/normalize"
)

// Result is one ranked address match.
type Result struct {
	Address      string  `json:"address" example:"1110 Wien, Gasometer" doc:"PostalCode Municipality, StreetName"`
	StreetNumber string  `json:"street_number" example:"A/1 3.Stock 22b" doc:"Trailing part of the query that was trimmed off"`
	Score        float64 `json:"score" minimum:"0" maximum:"1" example:"0.95" doc:"Matching trigrams / unique query trigrams"`
}

// Search evaluates the full query and every right-to-left trim at word boundaries in
// a single pass. The split is chosen by the best document's net evidence
// (matched - missed trigrams): a pure ratio would always favour trimming down to the
// shortest exact prefix, while net evidence only drops words that mostly miss.
// All documents are then scored against that split:
// Score = matching trigrams / unique trigrams in the trimmed query.
func (ix *Index) Search(q string, limit int, minScore float64) []Result {
	p := makePlan(q)
	nw := len(p.words)
	if nw == 0 || limit <= 0 {
		return []Result{}
	}
	sc := ix.pool.Get().(*scratch)
	defer func() { sc.reset(); ix.pool.Put(sc) }()
	st := sc.st

	groupEnd := make([]int, nw) // number of query trigrams in splits 0..g
	ki := 0
	for g := 0; g < nw; g++ {
		start := ki
		for ; ki < len(p.keys) && p.groupOf[ki] == g; ki++ {
			for _, d := range ix.postings[p.keys[ki]] {
				s := &st[d]
				if s.count == 0 {
					sc.touched = append(sc.touched, d)
				}
				s.count++
				if s.mark != uint8(g+1) {
					s.mark = uint8(g + 1)
					sc.group = append(sc.group, d)
				}
			}
		}
		groupEnd[g] = ki
		if ki == start {
			continue
		}
		// Docs not hit in this group only lose net evidence, so only hit docs can improve.
		for _, d := range sc.group {
			s := &st[d]
			if net := int16(2*int(s.count) - ki); s.bestK1 == 0 || net >= s.bestNet {
				s.bestNet, s.bestK1 = net, uint8(g+1)
			}
		}
		sc.group = sc.group[:0]
	}
	if len(sc.touched) == 0 {
		return []Result{}
	}

	top := sc.touched[0]
	for _, d := range sc.touched[1:] {
		if a, b := &st[d], &st[top]; a.bestNet > b.bestNet || a.bestNet == b.bestNet && ix.docLen[d] < ix.docLen[top] {
			top = d
		}
	}
	// A split adding no new trigrams is equivalent; prefer the longer address part.
	k := int(st[top].bestK1) - 1
	for k+1 < nw && groupEnd[k+1] == groupEnd[k] {
		k++
	}

	// Re-score every doc against the chosen split by removing the trimmed trigrams.
	for _, key := range p.keys[groupEnd[k]:] {
		for _, d := range ix.postings[key] {
			st[d].count--
		}
	}
	number := streetNumber(q, p.words[k])
	denom := float64(groupEnd[k])
	ids := ix.topK(sc, limit, int(math.Ceil(minScore*denom-1e-9)))
	out := make([]Result, len(ids))
	for i, d := range ids {
		out[i] = Result{Address: ix.docs[d], StreetNumber: number, Score: math.Round(float64(st[d].count)/denom*1e4) / 1e4}
	}
	return out
}

func streetNumber(q string, last normalize.Word) string {
	return strings.TrimSpace(strings.TrimLeftFunc(q[last.End:], normalize.IsSeparator))
}

// better orders by matched trigrams (the score numerator), then shorter doc, then id.
func (ix *Index) better(st []docState, a, b uint32) bool {
	if st[a].count != st[b].count {
		return st[a].count > st[b].count
	}
	if ix.docLen[a] != ix.docLen[b] {
		return ix.docLen[a] < ix.docLen[b]
	}
	return a < b
}

// topK selects the best `limit` docs with at least minCount matches using a bounded min-heap.
func (ix *Index) topK(sc *scratch, limit, minCount int) []uint32 {
	st := sc.st
	minCount = max(minCount, 1)
	h := make([]uint32, 0, limit)
	worse := func(i, j int) bool { return ix.better(st, h[j], h[i]) }
	for _, d := range sc.touched {
		if int(st[d].count) < minCount {
			continue
		}
		if len(h) < limit {
			h = append(h, d)
			for i := len(h) - 1; i > 0; {
				p := (i - 1) / 2
				if !worse(i, p) {
					break
				}
				h[i], h[p] = h[p], h[i]
				i = p
			}
			continue
		}
		if !ix.better(st, d, h[0]) {
			continue
		}
		h[0] = d
		for i := 0; ; {
			l, m := 2*i+1, i
			if l < len(h) && worse(l, m) {
				m = l
			}
			if r := l + 1; r < len(h) && worse(r, m) {
				m = r
			}
			if m == i {
				break
			}
			h[i], h[m] = h[m], h[i]
			i = m
		}
	}
	slices.SortFunc(h, func(a, b uint32) int {
		if ix.better(st, a, b) {
			return -1
		}
		return 1
	})
	return h
}
