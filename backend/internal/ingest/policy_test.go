package ingest_test

import (
	"testing"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

func TestEffectivePolicy(t *testing.T) {
	tests := []struct {
		configured db.TextPolicy
		reserved   bool
		want       db.TextPolicy
	}{
		{db.TextPolicyFullText, false, db.TextPolicyFullText},
		{db.TextPolicyFeedFields, false, db.TextPolicyFeedFields},
		{db.TextPolicyNone, false, db.TextPolicyNone},
		{db.TextPolicyFullText, true, db.TextPolicyNone},
		{db.TextPolicyFeedFields, true, db.TextPolicyNone},
		{db.TextPolicyNone, true, db.TextPolicyNone},
		{"", false, db.TextPolicyNone}, // unknown value fails closed
	}
	for _, tt := range tests {
		if got := ingest.EffectivePolicy(tt.configured, tt.reserved); got != tt.want {
			t.Errorf("EffectivePolicy(%q, %v) = %q, want %q", tt.configured, tt.reserved, got, tt.want)
		}
	}
}
