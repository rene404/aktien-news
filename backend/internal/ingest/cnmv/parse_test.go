package cnmv_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/cnmv"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

const (
	ip  = "https://www.cnmv.es/Portal/Informacion-Privilegiada/Resultado-IP.aspx?nreg="
	oir = "https://www.cnmv.es/Portal/Otra-Informacion-Relevante/Resultado-OIR.aspx?nreg="
)

// Live responses of 2026-11-08. Title = "issuer: subject", both as the CNMV
// publishes them ("fielmente, sin manipular"); time and category are dropped.
func TestParseInsideInformation(t *testing.T) {
	arts, err := cnmv.Parse(read(t, "ip.xml"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ingest.RawArticle{
		{
			ExternalID:  "3413",
			URL:         ip + "3413",
			Title:       "ACS, ACTIVIDADES DE CONSTRUCCION Y SERVICIOS, S.A.: El Grupo ACS niega la OPA sobre Hochtief y la venta de Turner",
			PublishedAt: time.Date(2026, 11, 7, 11, 37, 38, 0, time.UTC),
			IssuerHint:  "ACS, ACTIVIDADES DE CONSTRUCCION Y SERVICIOS, S.A.",
		},
		{
			ExternalID:  "3412",
			URL:         ip + "3412",
			Title:       "TUBOS REUNIDOS, S.A. (EN LIQUIDACIÓN): La CNMV informa que ha acordado la exclusión de negociación de las acciones de Tubos Reunidos, S.A. (en liquidación).",
			PublishedAt: time.Date(2026, 11, 6, 17, 0, 3, 0, time.UTC),
			IssuerHint:  "TUBOS REUNIDOS, S.A. (EN LIQUIDACIÓN)",
		},
	}
	if !reflect.DeepEqual(arts, want) {
		t.Errorf("got\n%+v\nwant\n%+v", arts, want)
	}
}

func TestParseOtherRelevantInformation(t *testing.T) {
	arts, err := cnmv.Parse(read(t, "oir.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("%d items, want 2", len(arts))
	}
	want := ingest.RawArticle{
		ExternalID:  "43537",
		URL:         oir + "43537",
		Title:       "AMPER, S.A.: Inscripción Escritura Ampliación de Capital adquisición Teltronic",
		PublishedAt: time.Date(2026, 11, 6, 17, 14, 44, 0, time.UTC),
		IssuerHint:  "AMPER, S.A.",
	}
	if arts[0] != want {
		t.Errorf("got %+v\nwant %+v", arts[0], want)
	}
}

func TestParseEdgeCases(t *testing.T) {
	arts, err := cnmv.Parse(read(t, "edge.xml"))
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, a := range arts {
		titles = append(titles, a.Title)
	}
	want := []string{
		"MAPFRE, S.A.: Resultados 3T & cartera <estable>", // HTML entities decoded
		"VIDRALA, S.A.: Sobre dividendos",                 // no subject: the category stands in
		// no link, unparseable date, no nreg: skipped
	}
	if !reflect.DeepEqual(titles, want) {
		t.Errorf("titles = %q, want %q", titles, want)
	}
}

func TestParseEmptyFeed(t *testing.T) {
	arts, err := cnmv.Parse(read(t, "empty.xml"))
	if err != nil || len(arts) != 0 {
		t.Fatalf("arts = %v, err = %v", arts, err)
	}
}

func TestParseBrokenFeedIsError(t *testing.T) {
	for _, b := range [][]byte{read(t, "broken.xml"), []byte("<html><body>Wartung</body></html>")} {
		if _, err := cnmv.Parse(b); err == nil {
			t.Errorf("%.40q: want error", b)
		}
	}
}
