package ingest

import "github.com/rene404/aktien-news/backend/internal/store/db"

// EffectivePolicy is the text policy that actually applies to a source: a
// detected TDM reservation forces none, whatever was configured. Unknown
// values fail closed.
func EffectivePolicy(configured db.TextPolicy, tdmReserved bool) db.TextPolicy {
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
