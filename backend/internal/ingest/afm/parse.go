// Package afm parses the AFM register of managers' transactions (Art. 19 MAR,
// "transacties leidinggevenden"), exported as one XML file with the whole
// history since 2005. Copying and distribution are allowed when AFM is named
// as the source; the source is called "AFM – Transacties leidinggevenden".
//
// The title names the issuer only, never the person (docs/plan-eu-sources.md,
// decision 2). The export has no notification date: the transaction date
// stands in, at midnight in Amsterdam.
package afm

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // the worker image may have no zoneinfo

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

// DetailsURL + meldingid is the register page of one notification.
const DetailsURL = "https://www.afm.nl/en/sector/registers/meldingenregisters/transacties-leidinggevenden-mar19-/details?id="

// Window: only notifications whose transaction lies within 14 days before
// the newest one in the export. Measured against the export, not the clock,
// so the parser stays pure.
const Window = 14 * 24 * time.Hour

const title = "Eigengeschäft einer Führungskraft"

var amsterdam = mustLoad("Europe/Amsterdam")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type register struct {
	XMLName xml.Name `xml:"register"`
	Entries []entry  `xml:"vermelding"`
}

// entry deliberately has no fields for the person or their function.
type entry struct {
	ID     string `xml:"meldingid"`
	Date   string `xml:"transactiedatum"` // "11/5/2026 12:00:00 AM"
	Issuer string `xml:"uitgevendeinstelling"`
	LEI    string `xml:"lei"`
}

// Parse extracts the recent notifications. Entries without id, issuer or a
// readable date are skipped; a malformed export is an error.
func Parse(b []byte) ([]ingest.RawArticle, error) {
	var reg register
	if err := xml.Unmarshal(b, &reg); err != nil {
		return nil, fmt.Errorf("afm: decode export: %w", err)
	}

	type dated struct {
		e  entry
		at time.Time
	}
	var all []dated
	var newest time.Time
	for _, e := range reg.Entries {
		at, ok := parseDate(e.Date)
		if !ok || strings.TrimSpace(e.ID) == "" || strings.TrimSpace(e.Issuer) == "" {
			continue
		}
		all = append(all, dated{e, at})
		if at.After(newest) {
			newest = at
		}
	}

	arts := make([]ingest.RawArticle, 0, len(all))
	for _, d := range all {
		if newest.Sub(d.at) > Window {
			continue
		}
		id := strings.TrimSpace(d.e.ID)
		issuer := strings.TrimSpace(d.e.Issuer)
		arts = append(arts, ingest.RawArticle{
			ExternalID:  id,
			URL:         DetailsURL + id,
			Title:       issuer + ": " + title,
			PublishedAt: d.at.UTC(),
			IssuerHint:  issuer,
			IssuerLEI:   strings.ToUpper(strings.TrimSpace(d.e.LEI)),
		})
	}
	return arts, nil
}

// parseDate reads the US date in front of the (always midnight) time.
func parseDate(s string) (time.Time, bool) {
	day, _, _ := strings.Cut(strings.TrimSpace(s), " ")
	t, err := time.ParseInLocation("1/2/2006", day, amsterdam)
	return t, err == nil
}
