package parsers_test

import (
	"encoding/csv"
	"os"
	"slices"
	"testing"

	"github.com/rene404/aktien-news/backend/internal/ingest/parsers"
)

// Every source in the seed names a registered parser; otherwise the worker
// skips it with "no parser".
func TestSeedSourcesHaveParsers(t *testing.T) {
	f, err := os.Open("../../../seed/sources.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	col := slices.Index(rows[0], "parser")
	if col < 0 {
		t.Fatal("no parser column")
	}
	all := parsers.All()
	for _, r := range rows[1:] {
		if _, ok := all[r[col]]; !ok {
			t.Errorf("source %s: parser %q not registered", r[0], r[col])
		}
	}
}
