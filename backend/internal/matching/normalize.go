package matching

import (
	"strings"
	"unicode"
)

var foldReplacer = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"á", "a", "à", "a", "â", "a", "ã", "a", "å", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ø", "o",
	"ú", "u", "ù", "u", "û", "u",
	"ç", "c", "ñ", "n",
)

// Fold lowercases, transliterates umlauts/accents (ö → oe, é → e) and
// turns every run of non-alphanumerics into one space, so names compare
// independent of spelling variants like "Österreichische"/"Oesterreichische".
func Fold(s string) string {
	s = foldReplacer.Replace(strings.ToLower(s))
	var b strings.Builder
	space := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(b.String())
}

// Folded legal-form tokens. "co" covers "& Co." in "GmbH & Co. KGaA".
var legalForms = map[string]bool{
	"ag": true, "aktiengesellschaft": true, "se": true, "kgaa": true, "gmbh": true,
	"co": true, "kg": true, "sa": true, "s": true, "a": true, "nv": true, "n": true,
	"v": true, "plc": true, "inc": true, "corp": true, "corporation": true,
	"ltd": true, "limited": true, "llc": true, "spa": true, "ab": true, "asa": true,
}

// stripLegalForm drops trailing legal-form tokens from a folded name:
// "siemens aktiengesellschaft" → "siemens", "befesa s a" → "befesa".
func stripLegalForm(folded string) string {
	words := strings.Fields(folded)
	for len(words) > 1 && legalForms[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}

func hasLegalForm(folded string) bool {
	return stripLegalForm(folded) != folded
}
