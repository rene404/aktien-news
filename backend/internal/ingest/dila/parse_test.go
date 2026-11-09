package dila_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/dila"
)

const pdf = "https://fr.ftp.opendatasoft.com/datadila/INFOFI/"

// bundle is what the fetcher hands the parsers for the two fixture archives.
func bundle(t *testing.T) []byte {
	t.Helper()
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	f, _ := newFetcher(t, &now)
	b, err := f.Fetch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func utc(s string) time.Time {
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return ts.UTC()
}

// Real notices of 2026-11-06. Issuers publish French and English versions as
// separate notices; the English one is kept. Inside information only.
func TestParseAdhoc(t *testing.T) {
	arts, err := dila.ParseAdhoc(bundle(t))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			// A placeholder title from the wire: the issuer goes before it.
			ExternalID:  "BWR_170155_20261106",
			URL:         pdf + "BWR/2026/11/FCBWR170155_20261106.pdf",
			Title:       "GETLINK SE: Inside Information / Other news releases",
			PublishedAt: utc("2026-11-06T07:00:00Z"),
			IssuerHint:  "GETLINK SE",
			IssuerISIN:  "FR0010533075",
			IssuerLEI:   "9695007ZEQ7M0OE74G82",
		},
		{
			ExternalID:  "MKW_112166_20261106",
			URL:         pdf + "MKW/2026/11/FCMKW112166_20261106.pdf",
			Title:       "Kering: Martine Rose appointed Creative Director of Balenciaga",
			PublishedAt: utc("2026-11-06T07:00:00Z"),
			IssuerHint:  "KERING",
			IssuerISIN:  "FR0000121485",
			IssuerLEI:   "549300VGEJKB7SVUZR78",
		},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

// Every other notice type; corrections (REC) are skipped for now.
func TestParseFilings(t *testing.T) {
	arts, err := dila.ParseFilings(bundle(t))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, a := range arts {
		got = append(got, a.ExternalID+" | "+a.Title+" | "+a.PublishedAt.Format(time.RFC3339))
	}
	want := []string{
		"ACT_082250_20261106 | BASTIDE LE CONFORT MEDICAL: 2026.11.06 - BLCM - Avis de réunion valant avis de convocation à l'Assemblée générale mixte du 15 décembre 2026 | 2026-11-06T08:45:00Z",
		"MKW_112185_20261106 | HighCo: SHAREHOLDING | 2026-11-06T08:16:02Z",
		"BWR_170177_20261106 | EUROFINS SCIENTIFIC SE: Total number of voting rights and capital / Information on the total number of voting rights and the number of shares making up the capital | 2026-11-06T14:25:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	if arts[2].IssuerLEI != "529900JEHFM47DYY3S57" || arts[2].IssuerISIN != "FR0014000MR3" {
		t.Errorf("eurofins identifiers: %+v", arts[2])
	}
}

func TestParseEmptyBundle(t *testing.T) {
	for _, parse := range []ingest.Parser{dila.ParseAdhoc, dila.ParseFilings} {
		arts, err := parse([]byte(`<?xml version="1.0"?><Bundle></Bundle>`))
		if err != nil || len(arts) != 0 {
			t.Errorf("arts = %v, err = %v", arts, err)
		}
	}
}

func TestParseBrokenBundleIsError(t *testing.T) {
	for _, b := range []string{"<html>", `<Bundle><UniteInformation`} {
		if _, err := dila.ParseAdhoc([]byte(b)); err == nil {
			t.Errorf("%q: want error", b)
		}
	}
}
