package dila_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest/dila"
)

const dir = "https://echanges.dila.gouv.fr/OPENDATA/AMF/"

type fakeGetter struct {
	bodies map[string][]byte
	calls  map[string]int
}

func (f *fakeGetter) Fetch(_ context.Context, url string) ([]byte, error) {
	f.calls[url]++
	if b, ok := f.bodies[url]; ok {
		return b, nil
	}
	return nil, errors.New("unexpected url " + url)
}

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func newFetcher(t *testing.T, now *time.Time) (*dila.Fetcher, *fakeGetter) {
	t.Helper()
	g := &fakeGetter{
		bodies: map[string][]byte{
			dir:                                 read(t, "listing.html"),
			dir + "amf-gz_06112026_1000.tar.gz": read(t, "amf-gz_06112026_1000.tar.gz"),
			dir + "amf-gz_06112026_1600.tar.gz": read(t, "amf-gz_06112026_1600.tar.gz"),
			// amf-gz_05112026_1905.tar.gz is older than the window: never asked for.
		},
		calls: map[string]int{},
	}
	return &dila.Fetcher{Get: g, Now: func() time.Time { return *now }}, g
}

// docs/plan-eu-sources.md, task 6: the archives of the last 26 hours, read once.
func TestFetchLoadsRecentArchivesOnce(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC) // window from 2026-11-06 08:00
	f, g := newFetcher(t, &now)
	ctx := context.Background()

	b, err := f.Fetch(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(b, []byte("<UniteInformation ")); n != 9 {
		t.Errorf("bundle holds %d notices, want 9 (7 + 2 XML files, no PDFs)", n)
	}
	if g.calls[dir+"amf-gz_05112026_1905.tar.gz"] != 0 {
		t.Error("loaded an archive older than 26 hours")
	}

	// The second source half an hour later: same notices, nothing downloaded again.
	now = now.Add(30 * time.Minute)
	b2, err := f.Fetch(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, b2) {
		t.Error("second fetch returned a different bundle")
	}
	for _, name := range []string{"amf-gz_06112026_1000.tar.gz", "amf-gz_06112026_1600.tar.gz"} {
		if c := g.calls[dir+name]; c != 1 {
			t.Errorf("%s downloaded %d times, want 1", name, c)
		}
	}

	// A day later both archives fall out of the window and out of memory.
	now = now.Add(27 * time.Hour)
	b3, err := f.Fetch(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b3, []byte("<UniteInformation ")) {
		t.Error("archives outside the window are still in the bundle")
	}
}

func TestFetchBrokenArchiveIsError(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	f, g := newFetcher(t, &now)
	g.bodies[dir] = []byte(strings.Replace(string(read(t, "listing.html")), "amf-gz_06112026_1600.tar.gz", "amf-gz_06112026_1700.tar.gz", -1))
	g.bodies[dir+"amf-gz_06112026_1700.tar.gz"] = read(t, "amf-gz_broken.tar.gz")
	if _, err := f.Fetch(context.Background(), dir); err == nil || !strings.Contains(err.Error(), "amf-gz_06112026_1700.tar.gz") {
		t.Fatalf("err = %v, want an error naming the broken archive", err)
	}
}

func TestFetchListingFailureIsError(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	f, g := newFetcher(t, &now)
	delete(g.bodies, dir)
	if _, err := f.Fetch(context.Background(), dir); err == nil {
		t.Fatal("want error")
	}
}

// One malformed notice must not block the others for a day; Latin-1 is read.
func TestFetchSkipsMalformedNoticesAndReadsLatin1(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	f, g := newFetcher(t, &now)
	good := `<?xml version="1.0"?>` + "\n" + `<UniteInformation UIN_IDT_UIN="000001_20261106"><identificationSociete ISO_NOM_SOC="SOCI` + "\xc9" + `T` + "\xc9" + ` G` + "\xc9" + `N` + "\xc9" + `RALE"/></UniteInformation>`
	g.bodies[dir+"amf-gz_06112026_1600.tar.gz"] = tarGz(t, map[string]string{
		"xml/X/2026/11/MDX000001_20261106.xml": good,
		"xml/X/2026/11/MDX000002_20261106.xml": `<UniteInformation UIN_IDT_UIN="0`,
	})
	b, err := f.Fetch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "SOCIÉTÉ GÉNÉRALE") || strings.Contains(string(b), "000002") {
		t.Errorf("bundle:\n%s", b)
	}
}

func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(files[name]))
	}
	tw.Close()
	zw.Close()
	return buf.Bytes()
}
