package index

import "asv6/internal/normalize"

const maxWords = 32

// plan is a preprocessed query. Split candidate g keeps words[0..g] and moves
// the rest into the street number. Because the canonical prefix is followed by a
// space in the full padded string, trigrams of candidate g are exactly the keys
// with groupOf <= g (keys are ordered by first occurrence).
type plan struct {
	words   []normalize.Word
	keys    []uint32 // unique query trigrams in first-occurrence order
	groupOf []int    // split group of each key
}

func makePlan(q string) plan {
	words := normalize.Words(q)
	if len(words) > maxWords {
		words = words[:maxWords]
	}
	p := plan{words: words}
	if len(words) == 0 {
		return p
	}
	runes := []rune{' '}
	ends := make([]int, len(words)) // index of the space following word i
	for i, w := range words {
		if i > 0 {
			runes = append(runes, ' ')
		}
		runes = append(runes, []rune(w.Norm)...)
		ends[i] = len(runes)
	}
	runes = append(runes, ' ')

	seen := make(map[uint32]struct{}, len(runes))
	g := 0
	for s := 0; s+2 < len(runes); s++ {
		for g < len(words)-1 && s > ends[g]-2 {
			g++
		}
		k := key(runes[s], runes[s+1], runes[s+2])
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		p.keys = append(p.keys, k)
		p.groupOf = append(p.groupOf, g)
	}
	return p
}
