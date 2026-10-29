package dedupe

import (
	"hash/fnv"
	"math/bits"
	"slices"
	"strings"
	"unicode"
)

// MaxDistance is the largest Hamming distance between two title
// fingerprints that may still be the same story.
const MaxDistance = 3

// SimHash is a 64-bit similarity fingerprint of a title. Features are the
// lowercased word tokens and adjacent token pairs; titles that share most of
// them end up a small Hamming distance apart.
func SimHash(title string) uint64 {
	tokens := words(title)
	var sums [64]int
	add := func(feature string) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(feature))
		v := h.Sum64()
		for i := range sums {
			if v&(1<<i) != 0 {
				sums[i]++
			} else {
				sums[i]--
			}
		}
	}
	for i, tok := range tokens {
		add(tok)
		if i > 0 {
			add(tokens[i-1] + " " + tok)
		}
	}
	var out uint64
	for i, s := range sums {
		if s > 0 {
			out |= 1 << i
		}
	}
	return out
}

// Hamming is the number of differing bits between two fingerprints.
func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// SameStory decides whether two titles report the same story: their
// fingerprints are at most MaxDistance apart and they carry the same numbers.
// The number rule keeps follow-ups apart that differ only in a date, quarter
// or paragraph ("Korrektur vom 23.10." vs "vom 27.10."), which SimHash alone
// cannot do on long titles.
func SameStory(titleA string, hashA uint64, titleB string, hashB uint64) bool {
	return Hamming(hashA, hashB) <= MaxDistance && slices.Equal(numbers(titleA), numbers(titleB))
}

func words(title string) []string {
	return strings.FieldsFunc(strings.ToLower(title), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// numbers returns the title's all-digit tokens, sorted.
func numbers(title string) []string {
	var out []string
	for _, w := range words(title) {
		if strings.IndexFunc(w, func(r rune) bool { return !unicode.IsDigit(r) }) < 0 {
			out = append(out, w)
		}
	}
	slices.Sort(out)
	return out
}
