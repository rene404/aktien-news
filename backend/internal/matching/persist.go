package matching

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/rene404/aktien-news/backend/internal/store"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// LoadIndex builds an Index from the catalog tables.
func LoadIndex(ctx context.Context, conn db.DBTX) (*Index, error) {
	q := db.New(conn)
	companies, err := q.IndexCompanies(ctx)
	if err != nil {
		return nil, fmt.Errorf("load companies: %w", err)
	}
	aliases, err := q.IndexAliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aliases: %w", err)
	}
	listings, err := q.IndexListings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load listings: %w", err)
	}

	byID := make(map[int64]*CompanyEntry, len(companies))
	entries := make([]CompanyEntry, len(companies))
	for i, c := range companies {
		entries[i] = CompanyEntry{ID: c.ID, Name: c.Name, LegalName: deref(c.LegalName), CIK: deref(c.Cik), Uncurated: !c.Curated}
		byID[c.ID] = &entries[i]
	}
	for _, a := range aliases {
		if e := byID[a.CompanyID]; e != nil {
			e.Aliases = append(e.Aliases, Alias{Text: a.Alias, Kind: a.Kind})
		}
	}
	for _, l := range listings {
		if e := byID[l.CompanyID]; e != nil {
			e.Listings = append(e.Listings, Listing{Ticker: l.Ticker, MIC: l.Mic, ISIN: deref(l.Isin)})
		}
	}
	return NewIndex(entries), nil
}

// Link is the current link state of one company for an article.
type Link struct {
	CompanyID  int64
	Status     string
	Confidence float64
	Method     string
}

func CurrentLinks(ctx context.Context, conn db.DBTX, articleID int64) ([]Link, error) {
	rows, err := db.New(conn).LatestEntities(ctx, articleID)
	if err != nil {
		return nil, err
	}
	links := make([]Link, len(rows))
	for i, r := range rows {
		links[i] = Link{CompanyID: r.CompanyID, Status: string(r.Status), Confidence: r.Confidence, Method: string(r.Method)}
	}
	return links, nil
}

// Persist records matches for an article. The table is append-only, so a
// row is only added when the state actually changes:
//   - no previous row          → insert (linked or pending)
//   - pending, now linked      → insert linked
//   - rejected (by an admin)   → never touched again
//   - linked                   → never downgraded
//
// Calls for the same article are serialised with an advisory lock. Given a
// pgx.Tx it runs as a savepoint inside the caller's transaction.
func Persist(ctx context.Context, conn store.TxBeginner, articleID int64, matches []Match) error {
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := q.LockArticle(ctx, articleID); err != nil {
			return fmt.Errorf("lock article %d: %w", articleID, err)
		}
		latest, err := q.LatestEntities(ctx, articleID)
		if err != nil {
			return err
		}
		state := make(map[int64]db.LinkStatus, len(latest))
		for _, l := range latest {
			state[l.CompanyID] = l.Status
		}

		for _, m := range matches {
			status := StatusFor(m.Confidence)
			if status == "" {
				continue
			}
			prev, seen := state[m.CompanyID]
			upgrade := prev == db.LinkStatusPending && status == "linked"
			if seen && !upgrade {
				continue
			}
			if err := q.InsertEntity(ctx, db.InsertEntityParams{
				ArticleID:  articleID,
				CompanyID:  m.CompanyID,
				Confidence: m.Confidence,
				Method:     db.MatchMethod(m.Method),
				Status:     db.LinkStatus(status),
			}); err != nil {
				return fmt.Errorf("insert link article=%d company=%d: %w", articleID, m.CompanyID, err)
			}
			state[m.CompanyID] = db.LinkStatus(status)
		}
		return nil
	})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
