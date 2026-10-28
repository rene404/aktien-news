package dedupe_test

import (
	"bytes"
	"testing"

	"github.com/rene404/aktien-news/backend/internal/dedupe"
)

func TestCanonicalURL(t *testing.T) {
	const want = "https://finanznachrichten.de/nachrichten-2026-10/123-x.htm"
	same := []string{
		"https://www.finanznachrichten.de/nachrichten-2026-10/123-x.htm",
		"http://www.finanznachrichten.de/nachrichten-2026-10/123-x.htm",
		"HTTPS://WWW.FinanzNachrichten.de:443/nachrichten-2026-10/123-x.htm",
		"https://www.finanznachrichten.de/nachrichten-2026-10/123-x.htm?utm_source=rss&utm_medium=feed",
		"https://www.finanznachrichten.de/nachrichten-2026-10/123-x.htm#top",
		"https://www.finanznachrichten.de/nachrichten-2026-10/123-x.htm?fbclid=abc&gclid=def",
	}
	for _, in := range same {
		got, err := dedupe.CanonicalURL(in)
		if err != nil {
			t.Errorf("CanonicalURL(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("CanonicalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCanonicalURLKeepsMeaningfulQueryInStableOrder(t *testing.T) {
	a, _ := dedupe.CanonicalURL("https://www.ots.at/presseaussendung?id=2&lang=de&utm_campaign=x")
	b, _ := dedupe.CanonicalURL("https://ots.at/presseaussendung?lang=de&id=2")
	if a != b || a != "https://ots.at/presseaussendung?id=2&lang=de" {
		t.Errorf("got %q and %q", a, b)
	}
	c, _ := dedupe.CanonicalURL("https://ots.at/presseaussendung?id=3&lang=de")
	if c == a {
		t.Error("different id must not collapse")
	}
}

func TestCanonicalURLTrailingSlash(t *testing.T) {
	a, _ := dedupe.CanonicalURL("https://example.com/news/")
	b, _ := dedupe.CanonicalURL("https://example.com/news")
	root, _ := dedupe.CanonicalURL("https://example.com/")
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
	if root != "https://example.com/" {
		t.Errorf("root = %q", root)
	}
}

func TestCanonicalURLRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "not a url", "ftp://example.com/x", "/relative/path"} {
		if _, err := dedupe.CanonicalURL(in); err == nil {
			t.Errorf("CanonicalURL(%q): want error", in)
		}
	}
}

func TestTitleHash(t *testing.T) {
	h1 := dedupe.TitleHash("EQS-Adhoc: Lenzing AG:  Prognose  angehoben")
	h2 := dedupe.TitleHash("  eqs-adhoc: LENZING AG: Prognose angehoben ")
	h3 := dedupe.TitleHash("EQS-Adhoc: Lenzing AG: Prognose gesenkt")
	if !bytes.Equal(h1, h2) {
		t.Error("case/whitespace variants must hash equal")
	}
	if bytes.Equal(h1, h3) {
		t.Error("different titles must hash differently")
	}
	if len(h1) != 32 {
		t.Errorf("hash length = %d, want 32 (sha256)", len(h1))
	}
}
