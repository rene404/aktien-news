package ingest_test

import (
	"testing"

	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

func TestTitlesMayGoToLLM(t *testing.T) {
	for _, p := range []db.TextPolicy{db.TextPolicyNone, db.TextPolicyFeedFields, db.TextPolicyFullText} {
		if !ingest.TitlesToLLM(p) {
			t.Errorf("%s: titles should go to the LLM", p)
		}
	}
	if ingest.TitlesToLLM(db.TextPolicyNoLlm) {
		t.Error("no_llm: titles must not go to the LLM")
	}
}

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
		// no_llm: not even titles go to the LLM (FCA NSM, plan decision 3);
		// a TDM reservation never loosens it.
		{db.TextPolicyNoLlm, false, db.TextPolicyNoLlm},
		{db.TextPolicyNoLlm, true, db.TextPolicyNoLlm},
	}
	for _, tt := range tests {
		if got := ingest.EffectivePolicy(tt.configured, tt.reserved); got != tt.want {
			t.Errorf("EffectivePolicy(%q, %v) = %q, want %q", tt.configured, tt.reserved, got, tt.want)
		}
	}
}
