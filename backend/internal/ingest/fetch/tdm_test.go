package fetch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rene404/aktien-news/backend/internal/ingest/fetch"
)

// tdmSite serves the feed at /rss/finanzen plus whatever files and headers a
// case configures; every other path is a 404.
type tdmSite struct {
	header string            // TDM-Reservation header on the feed
	files  map[string]string // path -> body
	status map[string]int    // path -> forced status
}

func (s tdmSite) start(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code, ok := s.status[r.URL.Path]; ok {
			w.WriteHeader(code)
			return
		}
		if r.URL.Path == "/rss/finanzen" {
			if s.header != "" {
				w.Header().Set("TDM-Reservation", s.header)
			}
			_, _ = w.Write([]byte("<rss/>"))
			return
		}
		if body, ok := s.files[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckTDM(t *testing.T) {
	ots, err := os.ReadFile("testdata/ots_tdmrep.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		site    tdmSite
		want    bool
		wantErr bool
	}{
		{name: "root file with reservation (OTS)", site: tdmSite{files: map[string]string{"/tdmrep.json": string(ots)}}, want: true},
		{name: "well-known file with reservation", site: tdmSite{files: map[string]string{"/.well-known/tdmrep.json": `[{"location":"/rss/*","tdm-reservation":1}]`}}, want: true},
		{name: "header", site: tdmSite{header: "1"}, want: true},
		{name: "header 0", site: tdmSite{header: "0"}, want: false},
		{name: "no files (404)", site: tdmSite{}, want: false},
		{name: "reservation for another path", site: tdmSite{files: map[string]string{"/.well-known/tdmrep.json": `[{"location":"/premium/*","tdm-reservation":1}]`}}, want: false},
		{name: "explicitly not reserved", site: tdmSite{files: map[string]string{"/.well-known/tdmrep.json": `[{"location":"/","tdm-reservation":0}]`}}, want: false},
		{name: "anchored location does not match", site: tdmSite{files: map[string]string{"/tdmrep.json": `[{"location":"/$","tdm-reservation":1}]`}}, want: false},
		{name: "malformed file", site: tdmSite{files: map[string]string{"/tdmrep.json": `{not json`}}, wantErr: true},
		{name: "server error on file", site: tdmSite{status: map[string]int{"/.well-known/tdmrep.json": http.StatusInternalServerError}}, wantErr: true},
		{name: "well-known forbidden, root file reserved", site: tdmSite{
			status: map[string]int{"/.well-known/tdmrep.json": http.StatusForbidden},
			files:  map[string]string{"/tdmrep.json": string(ots)},
		}, want: true},
		{name: "well-known soft 404 page, root file reserved", site: tdmSite{
			files: map[string]string{"/.well-known/tdmrep.json": "<html>not found</html>", "/tdmrep.json": string(ots)},
		}, want: true},
		{name: "well-known fails, root missing", site: tdmSite{
			status: map[string]int{"/.well-known/tdmrep.json": http.StatusInternalServerError},
		}, wantErr: true},
		{name: "feed HEAD not allowed still checks files", site: tdmSite{
			status: map[string]int{"/rss/finanzen": http.StatusMethodNotAllowed},
			files:  map[string]string{"/tdmrep.json": string(ots)},
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := tt.site.start(t)
			got, err := fetch.New("ua", 5*time.Second).CheckTDM(context.Background(), srv.URL+"/rss/finanzen")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("reserved = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckTDMNetworkErrorIsError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL + "/rss/finanzen"
	srv.Close()

	if _, err := fetch.New("ua", 5*time.Second).CheckTDM(context.Background(), url); err == nil {
		t.Fatal("unreachable host: want error")
	}
}
