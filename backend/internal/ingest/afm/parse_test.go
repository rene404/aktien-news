package afm_test

import (
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/afm"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

const details = "https://www.afm.nl/en/sector/registers/meldingenregisters/transacties-leidinggevenden-mar19-/details?id="

// Real entries of the export of 2026-11-08, person names replaced (we never
// use them). The window is the 14 days before the newest entry; the date is
// the transaction date (US format) at midnight in Amsterdam.
func TestParse(t *testing.T) {
	arts, err := afm.Parse(read(t, "dd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	art := func(id, issuer, lei string, at time.Time) ingest.RawArticle {
		return ingest.RawArticle{
			ExternalID:  id,
			URL:         details + id,
			Title:       issuer + ": Eigengeschäft einer Führungskraft",
			PublishedAt: at,
			IssuerHint:  issuer,
			IssuerLEI:   lei,
		}
	}
	want := []ingest.RawArticle{
		art("2026110389A7BD17-C37D-4072-8996-EF9A09F8D9C1", "STMicroelectronics N.V.", "213800Z8NOHIKRI42W10", time.Date(2026, 11, 2, 23, 0, 0, 0, time.UTC)),
		art("2026102896724A86-8AB2-4ECD-B879-54D4C10D82D7", "Universal Music Group N.V.", "724500GJBUL3D9TW9Y18", time.Date(2026, 10, 27, 23, 0, 0, 0, time.UTC)),
		art("202610272B138DA4-C65E-4C20-8377-4AB813042063", "Universal Music Group N.V.", "724500GJBUL3D9TW9Y18", time.Date(2026, 10, 26, 23, 0, 0, 0, time.UTC)),
		art("2026110536B2B2B2-9670-4417-8D87-1A85BD2EF00C", "argenx SE", "7245009C5FZE6G9ODQ71", time.Date(2026, 11, 4, 23, 0, 0, 0, time.UTC)),
		// Still summer time in Amsterdam (UTC+2).
		art("2026102308DDD702-BD6B-49F2-979F-5B7C38B8A061", "NN Group N.V.", "724500OHYNDT9OY6Q215", time.Date(2026, 10, 22, 22, 0, 0, 0, time.UTC)),
		// Unilever 2022 and Qiagen 2005 are older than the window.
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseTitleNamesNoPerson(t *testing.T) {
	arts, err := afm.Parse(read(t, "dd.xml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range arts {
		for _, s := range []string{a.Title, a.IssuerHint, a.URL} {
			if regexpPerson.MatchString(s) {
				t.Errorf("person in %q", s)
			}
		}
	}
}

func TestParseEmpty(t *testing.T) {
	arts, err := afm.Parse(read(t, "empty.xml"))
	if err != nil || len(arts) != 0 {
		t.Fatalf("arts = %v, err = %v", arts, err)
	}
}

func TestParseBrokenIsError(t *testing.T) {
	for _, b := range [][]byte{read(t, "broken.xml"), []byte("<html><body>Onderhoud</body></html>")} {
		if _, err := afm.Parse(b); err == nil {
			t.Errorf("%.40q: want error", b)
		}
	}
}

var regexpPerson = regexp.MustCompile(`Person ?\d`)
