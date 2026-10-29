package fetch

import "testing"

func TestLocationMatches(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"", "/rss/finanzen", true},
		{"/", "/rss/finanzen", true},
		{"/", "", true},
		{"/rss/", "/rss/finanzen", true},
		{"/rss/*", "/rss/finanzen", true},
		{"/*/finanzen", "/rss/finanzen", true},
		{"/*.xml$", "/feeds/a.xml", true},
		{"/*.xml$", "/feeds/a.xml/more", false},
		{"/$", "/", true},
		{"/$", "/rss/finanzen", false},
		{"/premium/*", "/rss/finanzen", false},
		{"/rss/finanzen$", "/rss/finanzen", true},
		{"/ab*b$", "/ab", false},
	}
	for _, tt := range tests {
		if got := locationMatches(tt.pattern, tt.path); got != tt.want {
			t.Errorf("locationMatches(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}
