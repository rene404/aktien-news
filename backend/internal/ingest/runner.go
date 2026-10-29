package ingest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/dedupe"
	"github.com/rene404/aktien-news/backend/internal/matching"
	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// Fetcher retrieves a feed. ingest/fetch.HTTPFetcher is the live one.
type Fetcher interface {
	Fetch(ctx context.Context, url string) ([]byte, error)
}

// TDMChecker reports whether a source's site reserves TDM rights.
// ingest/fetch.HTTPFetcher is the live one.
type TDMChecker interface {
	CheckTDM(ctx context.Context, url string) (bool, error)
}

// TDMCheckInterval is how often a source's TDM reservation is re-checked.
const TDMCheckInterval = 24 * time.Hour

// Parser is a pure function from feed bytes to articles.
type Parser func([]byte) ([]RawArticle, error)

// Runner executes fetch → parse → dedupe → insert → match for due sources.
type Runner struct {
	Pool    *pgxpool.Pool
	Fetcher Fetcher
	Parsers map[string]Parser // by sources.parser
	TDM     TDMChecker        // nil disables the TDM check
	Now     func() time.Time
}

type result struct {
	status  db.RunStatus
	fetched int
	new     int
	err     error
}

// RunDue runs every enabled source whose fetch interval has elapsed. One
// failing source never stops the others; its run is recorded as failed and
// logged at WARNING. The returned error is only for infrastructure failures
// (database unreachable, catalog not loadable).
func (r *Runner) RunDue(ctx context.Context) error {
	q := db.New(r.Pool)
	sources, err := q.DueSources(ctx, r.Now())
	if err != nil {
		return fmt.Errorf("due sources: %w", err)
	}
	if len(sources) == 0 {
		return nil
	}
	index, err := matching.LoadIndex(ctx, r.Pool)
	if err != nil {
		return err
	}

	for _, src := range sources {
		if err := r.checkTDM(ctx, q, src); err != nil {
			return fmt.Errorf("tdm %s: %w", src.Key, err)
		}
		started := r.Now()
		runID, err := q.StartRun(ctx, db.StartRunParams{SourceID: src.ID, StartedAt: started})
		if err != nil {
			return fmt.Errorf("start run %s: %w", src.Key, err)
		}
		res := r.runSource(ctx, src, index)
		if res.err != nil {
			slog.Warn("ingest source failed", "source", src.Key, "status", res.status, "err", res.err)
		} else {
			slog.Info("ingest source done", "source", src.Key, "status", res.status, "fetched", res.fetched, "new", res.new)
		}
		if err := r.finish(ctx, q, runID, res); err != nil {
			return fmt.Errorf("finish run %s: %w", src.Key, err)
		}
	}
	return nil
}

// checkTDM refreshes the source's TDM reservation once per TDMCheckInterval.
// A failed check keeps the previous state and is retried on the next run;
// only database errors are returned.
func (r *Runner) checkTDM(ctx context.Context, q *db.Queries, src db.Source) error {
	now := r.Now()
	if r.TDM == nil || (src.TdmCheckedAt != nil && now.Sub(*src.TdmCheckedAt) < TDMCheckInterval) {
		return nil
	}
	reserved, err := r.TDM.CheckTDM(ctx, src.Url)
	if err != nil {
		slog.Warn("tdm check failed", "source", src.Key, "err", err)
		return nil
	}
	if reserved != src.TdmReserved {
		slog.Info("tdm reservation changed", "source", src.Key, "reserved", reserved)
	}
	return q.SetTDM(ctx, db.SetTDMParams{ID: src.ID, TdmReserved: reserved, TdmCheckedAt: &now})
}

func (r *Runner) runSource(ctx context.Context, src db.Source, index *matching.Index) (res result) {
	defer func() {
		if p := recover(); p != nil {
			res = result{status: db.RunStatusFailed, fetched: res.fetched, new: res.new, err: fmt.Errorf("panic: %v", p)}
		}
	}()

	parse, ok := r.Parsers[src.Parser]
	if !ok {
		return result{status: db.RunStatusSkipped, err: fmt.Errorf("no parser %q", src.Parser)}
	}
	body, err := r.Fetcher.Fetch(ctx, src.Url)
	if err != nil {
		return result{status: db.RunStatusFailed, err: err}
	}
	arts, err := parse(body)
	if err != nil {
		return result{status: db.RunStatusFailed, err: err}
	}

	res = result{status: db.RunStatusOk, fetched: len(arts)}
	var itemErrs []error
	for _, a := range arts {
		inserted, err := r.store(ctx, src, a, index)
		if err != nil {
			itemErrs = append(itemErrs, fmt.Errorf("%s: %w", a.URL, err))
			continue
		}
		if inserted {
			res.new++
		}
	}
	if len(itemErrs) > 0 {
		res.status = db.RunStatusPartial
		res.err = errors.Join(itemErrs...)
	}
	return res
}

// store inserts one article with its cluster and company links atomically. It reports
// false when the article was already known.
func (r *Runner) store(ctx context.Context, src db.Source, a RawArticle, index *matching.Index) (bool, error) {
	canonical, err := dedupe.CanonicalURL(a.URL)
	if err != nil {
		return false, err
	}
	hash := dedupe.SimHash(a.Title)
	inserted := false
	err = pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		art, err := q.InsertArticle(ctx, db.InsertArticleParams{
			SourceID:     src.ID,
			ExternalID:   a.ExternalID,
			Url:          a.URL,
			CanonicalUrl: canonical,
			Title:        a.Title,
			PublishedAt:  a.PublishedAt,
			ContentHash:  dedupe.TitleHash(a.Title),
			Simhash:      ptr(int64(hash)),
			IssuerHint:   optional(a.IssuerHint),
			IssuerIsin:   optional(a.IssuerISIN),
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // already known
		}
		if err != nil {
			return err
		}
		inserted = true
		if err := cluster(ctx, q, art.ID, art.Title, art.PublishedAt, hash, r.Now()); err != nil {
			return err
		}
		matches := index.Match(matching.Input{
			Title:           a.Title,
			IssuerHint:      a.IssuerHint,
			IssuerISIN:      a.IssuerISIN,
			ISINIsIssuer:    src.Kind == db.SourceKindAdhoc || src.Kind == db.SourceKindFiling,
			IssuerCompanyID: deref(src.IssuerCompanyID),
			IssuerCIK:       a.IssuerCIK,
		})
		return matching.Persist(ctx, tx, art.ID, matches)
	})
	return inserted, err
}

func (r *Runner) finish(ctx context.Context, q *db.Queries, runID int64, res result) error {
	finished := r.Now()
	status := res.status
	var msg *string
	if res.err != nil {
		s := res.err.Error()
		if len(s) > 2000 {
			s = s[:2000]
		}
		msg = &s
	}
	return q.FinishRun(ctx, db.FinishRunParams{
		ID: runID, FinishedAt: &finished, Status: &status,
		Fetched: int32(res.fetched), New: int32(res.new), Error: msg,
	})
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
