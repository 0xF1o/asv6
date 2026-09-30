// Package normalize turns free text into canonical, lowercase, ASCII-folded words.
package normalize

import (
	"strings"
	"unicode"
)

var fold = map[rune]string{
	'ä': "ae", 'ö': "oe", 'ü': "ue", 'ß': "ss", 'æ': "ae", 'œ': "oe",
	'á': "a", 'à': "a", 'â': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'ç': "c", 'ć': "c", 'č': "c", 'ď': "d", 'đ': "d",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ę': "e", 'ě': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i",
	'ł': "l", 'ľ': "l", 'ĺ': "l", 'ñ': "n", 'ń': "n", 'ň': "n",
	'ó': "o", 'ò': "o", 'ô': "o", 'õ': "o", 'ø': "o", 'ő': "o",
	'ř': "r", 'ŕ': "r", 'š': "s", 'ś': "s", 'ş': "s", 'ť': "t", 'ţ': "t",
	'ú': "u", 'ù': "u", 'û': "u", 'ů': "u", 'ű': "u", 'ý': "y", 'ÿ': "y",
	'ž': "z", 'ź': "z", 'ż': "z",
}

// Word is a run of letters/digits in the original input.
type Word struct {
	Start, End int    // byte offsets into the original string
	Norm       string // canonical form of the word
}

func isWordRune(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// IsSeparator reports whether r splits words (whitespace or punctuation).
func IsSeparator(r rune) bool { return !isWordRune(r) }

// Words splits s at whitespace/punctuation and canonicalizes each word.
func Words(s string) []Word {
	var words []Word
	var b strings.Builder
	start := -1
	flush := func(end int) {
		if start >= 0 {
			words = append(words, Word{Start: start, End: end, Norm: b.String()})
			b.Reset()
			start = -1
		}
	}
	for i, r := range s {
		if !isWordRune(r) {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
		}
		r = unicode.ToLower(r)
		if f, ok := fold[r]; ok {
			b.WriteString(f)
		} else {
			b.WriteRune(r)
		}
	}
	flush(len(s))
	return words
}

// Canonical returns the lowercase, transliterated, single-space-joined form of s.
func Canonical(s string) string {
	words := Words(s)
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = w.Norm
	}
	return strings.Join(parts, " ")
}
