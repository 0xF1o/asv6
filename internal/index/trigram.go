package index

import "hash/fnv"

// key maps a 3-rune window to a uint32. Runes < 0x400 are packed exactly into
// 30 bits; anything else is hashed into the space above 1<<30.
func key(a, b, c rune) uint32 {
	if a < 0x400 && b < 0x400 && c < 0x400 {
		return uint32(a)<<20 | uint32(b)<<10 | uint32(c)
	}
	h := fnv.New32a()
	h.Write([]byte(string([]rune{a, b, c})))
	return h.Sum32() | 0xC0000000
}

// pad returns " " + canonical + " " as runes.
func pad(canonical string) []rune {
	r := make([]rune, 0, len(canonical)+2)
	r = append(r, ' ')
	r = append(r, []rune(canonical)...)
	return append(r, ' ')
}

// trigrams appends all (possibly duplicate) trigram keys of padded runes to dst.
func trigrams(dst []uint32, padded []rune) []uint32 {
	for i := 0; i+2 < len(padded); i++ {
		dst = append(dst, key(padded[i], padded[i+1], padded[i+2]))
	}
	return dst
}
