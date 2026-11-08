package dila

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
)

// PDFBase serves the content file of a notice by its INF_FIC_NOM; it is the
// "url_de_recuperation" of info-financiere.gouv.fr. We only link to it.
const PDFBase = "https://fr.ftp.opendatasoft.com/datadila/INFOFI/"

// adhocTypes are the STYP codes of the AMF reference list (LOV) whose main
// label is "Informations privilégiées" (inside information, Art. 17 MAR):
// 001002 operations of the issuer, 001006 other releases, 001007 results,
// 001008 annual turnover, 015000 statements on rumours. Every other code is
// a filing (voting rights, reports, AGM documents, buy-backs …).
var adhocTypes = map[string]bool{"001002": true, "001006": true, "001007": true, "001008": true, "015000": true}

// twinWindow: issuers file the French and English version of a release as
// two notices, minutes apart.
const twinWindow = 10 * time.Minute

type bundle struct {
	XMLName xml.Name `xml:"Bundle"`
	Units   []unit   `xml:"UniteInformation"`
}

type unit struct {
	ID       string `xml:"UIN_IDT_UIN,attr"`
	Diffuser struct {
		Code string `xml:"IDI_COD_DIF,attr"`
	} `xml:"identificationDiffuseur"`
	Company struct {
		Name string `xml:"ISO_NOM_SOC,attr"`
		ISIN string `xml:"ISO_CD_ISI,attr"`
		LEI  string `xml:"ISO_CD_LEI,attr"`
	} `xml:"identificationSociete"`
	Info struct {
		Issued   string `xml:"INF_DAT_EMT,attr"`
		Type     string `xml:"INF_STP_PRI,attr"`
		Title    string `xml:"INF_TIT_INF,attr"`
		Language string `xml:"INF_LNG_INF,attr"`
		Update   struct {
			Status string `xml:"INF_UPG_STS,attr"`
		} `xml:"INF_UPG_INF"`
	} `xml:"InformationDeposee"`
	File struct {
		Name string `xml:"INF_FIC_NOM,attr"`
	} `xml:"FichierDeContenu"`
}

// ParseAdhoc keeps the inside information.
func ParseAdhoc(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(t string) bool { return adhocTypes[t] })
}

// ParseFilings keeps every other notice type.
func ParseFilings(b []byte) ([]ingest.RawArticle, error) {
	return parse(b, func(t string) bool { return !adhocTypes[t] })
}

func parse(b []byte, keep func(string) bool) ([]ingest.RawArticle, error) {
	var doc bundle
	if err := xml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("dila: decode bundle: %w", err)
	}
	if doc.XMLName.Local != "Bundle" {
		return nil, errors.New("dila: not a bundle")
	}

	arts := make([]ingest.RawArticle, 0, len(doc.Units))
	for _, u := range doc.Units {
		title := strings.TrimSpace(u.Info.Title)
		issuer := strings.TrimSpace(u.Company.Name)
		issued, err := time.Parse(time.RFC3339, strings.TrimSpace(u.Info.Issued))
		// Corrections (REC) and additions (ADD) refer to an earlier notice;
		// skipped for now.
		if !keep(u.Info.Type) || u.Info.Update.Status != "NEW" || u.ID == "" || u.Diffuser.Code == "" ||
			title == "" || issuer == "" || u.File.Name == "" || err != nil || hasEnglishTwin(u, issued, doc.Units) {
			continue
		}
		arts = append(arts, ingest.RawArticle{
			ExternalID:  u.Diffuser.Code + "_" + u.ID, // the counter is per diffuser
			URL:         PDFBase + strings.TrimSpace(u.File.Name),
			Title:       withIssuer(title, issuer),
			PublishedAt: issued.UTC(),
			IssuerHint:  issuer,
			IssuerISIN:  strings.ToUpper(strings.TrimSpace(u.Company.ISIN)),
			IssuerLEI:   strings.ToUpper(strings.TrimSpace(u.Company.LEI)),
		})
	}
	return arts, nil
}

// hasEnglishTwin reports whether u is a non-English notice whose English
// version (same issuer, type and diffuser, filed within minutes) is in the
// bundle too. Readers of the site understand English more widely.
func hasEnglishTwin(u unit, issued time.Time, all []unit) bool {
	if strings.EqualFold(u.Info.Language, "EN") {
		return false
	}
	for _, o := range all {
		if !strings.EqualFold(o.Info.Language, "EN") || o.Info.Type != u.Info.Type ||
			o.Diffuser.Code != u.Diffuser.Code || o.Company.Name != u.Company.Name || o.Info.Update.Status != "NEW" {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(o.Info.Issued))
		if err == nil && t.Sub(issued).Abs() <= twinWindow {
			return true
		}
	}
	return false
}

// withIssuer puts the issuer before a title that does not name it, as the
// OeKB parser does: "GETLINK SE: Inside Information / Other news releases".
func withIssuer(title, issuer string) string {
	first, _, _ := strings.Cut(issuer, " ")
	if strings.Contains(strings.ToLower(title), strings.ToLower(first)) {
		return title
	}
	return issuer + ": " + title
}
