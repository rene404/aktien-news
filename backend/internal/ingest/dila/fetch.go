// Package dila reads the AMF regulated information that the French DILA
// publishes as open data (Licence Ouverte) at echanges.dila.gouv.fr: every
// 1–3 hours an archive amf-gz_DDMMYYYY_HHMM.tar.gz with one XML metadata file
// and one content file (PDF) per notice.
//
// Fetching takes two steps, so it lives here instead of in ingest/fetch: the
// Fetcher reads the directory listing, downloads the archives of the last
// 26 hours and hands the parsers one XML bundle of their notices. The
// parsers (ParseAdhoc, ParseFilings) stay pure.
package dila

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

const (
	// Window is how far back archives are read. DILA publishes every 1–3
	// hours and the worker polls hourly; a day plus margin covers a pause.
	Window = 26 * time.Hour
	// listingTTL: both sources poll the same directory (2.6 MB of HTML)
	// within minutes; the second one reuses the first listing.
	listingTTL = 10 * time.Minute
	// maxXMLBytes caps one metadata file; they are a few KB.
	maxXMLBytes = 1 << 20
)

// archiveRe matches the archive names; the time in the name is taken as UTC
// (the listing shows them an hour later in Paris time).
var archiveRe = regexp.MustCompile(`href="(amf-gz_(\d{8}_\d{4})\.tar\.gz)"`)

// Fetcher loads the recent archives. Archives never change, so the XML
// files of each one are kept in memory, by name, while it is in the window;
// the two DILA sources share one Fetcher and download nothing twice.
type Fetcher struct {
	Get ingest.Fetcher // plain HTTP; must allow bodies of 20 MB
	Now func() time.Time

	mu        sync.Mutex
	listing   []byte
	listedAt  time.Time
	listedURL string
	archives  map[string][][]byte // archive name → its XML files
}

// Fetch returns the bundle of all notices in the archives of the last 26
// hours under dir (the directory URL, ending in "/").
func (f *Fetcher) Fetch(ctx context.Context, dir string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := f.Now()

	if f.listing == nil || f.listedURL != dir || now.Sub(f.listedAt) >= listingTTL {
		b, err := f.Get.Fetch(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("dila: listing: %w", err)
		}
		f.listing, f.listedAt, f.listedURL = b, now, dir
	}

	var recent []string
	for _, m := range archiveRe.FindAllStringSubmatch(string(f.listing), -1) {
		at, err := time.Parse("02012006_1504", m[2])
		if err != nil || now.Sub(at) > Window || at.After(now.Add(time.Hour)) {
			continue
		}
		recent = append(recent, m[1])
	}
	slices.SortFunc(recent, func(a, b string) int { return archiveTime(a).Compare(archiveTime(b)) })
	recent = slices.Compact(recent)

	if f.archives == nil {
		f.archives = map[string][][]byte{}
	}
	for name := range f.archives {
		if !slices.Contains(recent, name) {
			delete(f.archives, name)
		}
	}

	var out bytes.Buffer
	out.WriteString("<?xml version=\"1.0\"?>\n<Bundle>\n")
	for _, name := range recent {
		files, ok := f.archives[name]
		if !ok {
			b, err := f.Get.Fetch(ctx, dir+name)
			if err != nil {
				return nil, fmt.Errorf("dila: %s: %w", name, err)
			}
			if files, err = ExtractXML(b); err != nil {
				return nil, fmt.Errorf("dila: %s: %w", name, err)
			}
			files = usable(name, files)
			f.archives[name] = files
		}
		for _, x := range files {
			out.Write(x)
			out.WriteByte('\n')
		}
	}
	out.WriteString("</Bundle>\n")
	return out.Bytes(), nil
}

func archiveTime(name string) time.Time {
	m := archiveRe.FindStringSubmatch(`href="` + name + `"`)
	if m == nil {
		return time.Time{}
	}
	t, _ := time.Parse("02012006_1504", m[2])
	return t
}

// ExtractXML returns the metadata files (xml/…) of an archive, in archive
// order. Content files (PDFs) are skipped without being read into memory.
func ExtractXML(tgz []byte) ([][]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	defer zr.Close()
	tr := tar.NewReader(zr)
	var files [][]byte
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return files, nil
		}
		if err != nil {
			return nil, fmt.Errorf("untar: %w", err)
		}
		if h.Typeflag != tar.TypeReg || !strings.HasPrefix(h.Name, "xml/") || !strings.HasSuffix(h.Name, ".xml") {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, maxXMLBytes+1))
		if err != nil {
			return nil, fmt.Errorf("untar %s: %w", h.Name, err)
		}
		if len(b) > maxXMLBytes {
			return nil, fmt.Errorf("%s exceeds %d bytes", h.Name, maxXMLBytes)
		}
		files = append(files, b)
	}
}

// usable drops metadata files that are not well-formed XML, so one bad
// notice cannot block the whole bundle for a day, and turns Latin-1 into
// UTF-8 (the files declare no encoding).
func usable(archive string, files [][]byte) [][]byte {
	out := files[:0]
	for _, b := range files {
		if !utf8.Valid(b) {
			r := make([]rune, len(b))
			for i, c := range b {
				r[i] = rune(c)
			}
			b = []byte(string(r))
		}
		b = stripDecl(b)
		if err := wellFormed(b); err != nil {
			slog.Warn("dila: notice skipped", "archive", archive, "err", err)
			continue
		}
		out = append(out, b)
	}
	return out
}

func wellFormed(b []byte) error {
	d := xml.NewDecoder(bytes.NewReader(b))
	for {
		if _, err := d.Token(); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

var declRe = regexp.MustCompile(`^\s*<\?xml[^>]*\?>`)

func stripDecl(b []byte) []byte {
	return bytes.TrimSpace(declRe.ReplaceAll(b, nil))
}
