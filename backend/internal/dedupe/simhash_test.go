package dedupe_test

import (
	"testing"

	"github.com/rene404/aktien-news/backend/internal/dedupe"
)

func TestSimHashIsStableAndCaseInsensitive(t *testing.T) {
	a := dedupe.SimHash("EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen")
	b := dedupe.SimHash("eqs-cms:  ANDRITZ AG:  sonstige emittenten-/unternehmensinformationen ")
	if a != b {
		t.Errorf("%x != %x", a, b)
	}
	if a == 0 {
		t.Error("fingerprint of a real title is 0")
	}
}

func TestHamming(t *testing.T) {
	if got := dedupe.Hamming(0b1011, 0b0010); got != 2 {
		t.Errorf("Hamming = %d, want 2", got)
	}
}

// Real title pairs from OTS and FinanzNachrichten (October 2026).
func TestSameStory(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical release via OTS and FN",
			"EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen",
			"EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen", true},
		{"whitespace and case only",
			"EQS-News: ANDRITZ achieves high order intake and strong revenue growth in Q3 2026",
			"EQS-News: Andritz achieves high order intake and strong revenue growth in Q3 2026 ", true},
		{"correction of 23.10. vs 27.10. (German)",
			"EQS-PVR: KORREKTUR vom 23.10.2026: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 2 BörseG mit dem Ziel der europaweiten Verbreitung",
			"EQS-PVR: KORREKTUR vom 27.10.2026: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 2 BörseG mit dem Ziel der europaweiten Verbreitung", false},
		{"correction of 23 vs 27 October (English)",
			"EQS-PVR: Correction dated 23 October 2026: Lenzing AG: Release according to Article 135, Section 2 BörseG with the objective of Europe-wide distribution",
			"EQS-PVR: Correction dated 27 October 2026: Lenzing AG: Release according to Article 135, Section 2 BörseG with the objective of Europe-wide distribution", false},
		{"§ 135 Abs. 2 vs Abs. 3",
			"EQS-PVR: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 3 BörseG mit dem Ziel der europaweiten Verbreitung",
			"EQS-PVR: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 2 BörseG mit dem Ziel der europaweiten Verbreitung", false},
		{"same template, other issuer",
			"EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen",
			"EQS-CMS: Lenzing AG: Sonstige Emittenten-/Unternehmensinformationen", false},
		{"directors' dealings by different people",
			"EQS-DD: Lenzing AG: Christian Skilich, Kauf",
			"EQS-DD: Lenzing AG: Georg Kasperkovitz, Kauf", false},
		{"opposite news",
			"EQS-Adhoc: Lenzing AG: Prognose angehoben",
			"EQS-Adhoc: Lenzing AG: Prognose gesenkt", false},
		{"German and English version stay apart",
			"EQS-News: Lenzing Gruppe gibt Update zur Aktionärsstruktur nach Abschluss der Kapitalerhöhung",
			"EQS-News: Lenzing Group provides update on shareholder structure following completion of capital increase", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := dedupe.SameStory(tt.a, dedupe.SimHash(tt.a), tt.b, dedupe.SimHash(tt.b)); got != tt.want {
				t.Errorf("SameStory = %v, want %v (distance %d)", got, tt.want,
					dedupe.Hamming(dedupe.SimHash(tt.a), dedupe.SimHash(tt.b)))
			}
		})
	}
}
