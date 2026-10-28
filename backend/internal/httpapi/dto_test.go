package httpapi

import (
	"reflect"
	"strings"
	"testing"
)

// Legal invariant (design §8, AGENTS.md): public responses carry title,
// source, link, time and tickers — never text, snippets, sentiment or event
// types. This walks every public DTO, including nested types.
func TestPublicDTOsCarryNoTextOrScores(t *testing.T) {
	banned := []string{"body", "text", "snippet", "summary", "desc", "content", "teaser", "sentiment", "event", "score", "confidence"}

	seen := map[reflect.Type]bool{}
	var walk func(reflect.Type)
	walk = func(typ reflect.Type) {
		for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice {
			typ = typ.Elem()
		}
		if typ.Kind() != reflect.Struct || seen[typ] {
			return
		}
		seen[typ] = true
		for i := range typ.NumField() {
			f := typ.Field(i)
			names := strings.ToLower(f.Name + " " + f.Tag.Get("json"))
			for _, b := range banned {
				if strings.Contains(names, b) {
					t.Errorf("%s.%s (json %q) looks like %q", typ.Name(), f.Name, f.Tag.Get("json"), b)
				}
			}
			walk(f.Type)
		}
	}
	for _, dto := range publicDTOs {
		walk(reflect.TypeOf(dto))
	}
	if len(seen) < 5 {
		t.Fatalf("only %d DTO types checked; is publicDTOs complete?", len(seen))
	}
}
