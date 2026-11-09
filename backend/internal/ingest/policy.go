package ingest

import "github.com/rene404/aktien-news/backend/internal/store/db"

// EffectivePolicy is the text policy that actually applies to a source: a
// detected TDM reservation forces none, whatever was configured, but never
// loosens no_llm. Unknown values fail closed.
func EffectivePolicy(configured db.TextPolicy, tdmReserved bool) db.TextPolicy {
	if configured == db.TextPolicyNoLlm {
		return db.TextPolicyNoLlm
	}
	if tdmReserved {
		return db.TextPolicyNone
	}
	switch configured {
	case db.TextPolicyFeedFields, db.TextPolicyFullText:
		return configured
	default:
		return db.TextPolicyNone
	}
}

// TitlesToLLM reports whether the titles of a source with this (effective)
// policy may go to the LLM. Only no_llm keeps them out (plan-eu-sources,
// decision 3: the FCA declares a reservation for AI use).
func TitlesToLLM(p db.TextPolicy) bool {
	return p != db.TextPolicyNoLlm
}
