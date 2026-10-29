package ingest_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
	"github.com/rene404/aktien-news/backend/internal/store/testdb"
	"github.com/rene404/aktien-news/backend/seed"
)

const (
	urlAdhoc = "https://www.finanznachrichten.de/rss-aktien-adhoc/"
	urlDAX   = "https://www.finanznachrichten.de/rss-germany-40-nachrichten-1"
	urlATX   = "https://www.finanznachrichten.de/rss-austria-20-nachrichten-37"
	urlUSA   = "https://www.finanznachrichten.de/rss-nachrichten-aktien-usa"
	urlOTSF  = "https://www.ots.at/rss/finanzen"
	urlOTSW  = "https://www.ots.at/rss/wirtschaft"
	urlPRN   = "https://www.prnewswire.com/rss/news-releases-list.rss"
	urlVoest = "https://www.voestalpine.com/group/en/media/press-releases/rss"
)

type fakeFetcher struct {
	bodies map[string][]byte
	errs   map[string]error
	calls  map[string]int
}

func (f *fakeFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	f.calls[url]++
	if err := f.errs[url]; err != nil {
		return nil, err
	}
	if b, ok := f.bodies[url]; ok {
		return b, nil
	}
	return nil, errors.New("unexpected url " + url)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type clock struct{ now time.Time }

func (c *clock) Now() time.Time { return c.now }

func newRunner(t *testing.T) (*ingest.Runner, *pgxpool.Pool, *fakeFetcher, *clock) {
	t.Helper()
	ctx := context.Background()
	pool := testdb.New(t)
	if err := catalog.Seed(ctx, pool, seed.FS); err != nil {
		t.Fatal(err)
	}
	f := &fakeFetcher{
		bodies: map[string][]byte{
			urlAdhoc: readFixture(t, "finanznachrichten/testdata/adhoc.xml"),
			urlATX:   readFixture(t, "finanznachrichten/testdata/empty.xml"),
			urlUSA:   readFixture(t, "finanznachrichten/testdata/broken.xml"),
			urlOTSF:  readFixture(t, "pressrss/testdata/ots_finanzen.xml"),
			urlOTSW:  readFixture(t, "pressrss/testdata/empty.xml"),
			urlPRN:   readFixture(t, "pressrss/testdata/prn.xml"),
			urlVoest: readFixture(t, "pressrss/testdata/voestalpine_en.xml"),
		},
		errs:  map[string]error{urlDAX: errors.New("connection reset")},
		calls: map[string]int{},
	}
	c := &clock{now: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	r := &ingest.Runner{
		Pool:    pool,
		Fetcher: f,
		Parsers: map[string]ingest.Parser{
			"finanznachrichten": finanznachrichten.Parse,
			"pressrss":          pressrss.Parse,
		},
		Now: c.Now,
	}
	return r, pool, f, c
}

type runRow struct {
	key     string
	status  string
	fetched int
	new     int
	err     *string
}

func runs(t *testing.T, pool *pgxpool.Pool) map[string][]runRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT s.key, r.status::text, r.fetched, r.new, r.error
		FROM ingest_runs r JOIN sources s ON s.id = r.source_id ORDER BY r.id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string][]runRow{}
	for rows.Next() {
		var r runRow
		if err := rows.Scan(&r.key, &r.status, &r.fetched, &r.new, &r.err); err != nil {
			t.Fatal(err)
		}
		out[r.key] = append(out[r.key], r)
	}
	return out
}

func count(t *testing.T, pool *pgxpool.Pool, sql string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), sql).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRunDueIsolatesFailingSources(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}

	got := runs(t, pool)
	check := func(key, status string, fetched, new int) {
		t.Helper()
		if len(got[key]) != 1 {
			t.Fatalf("%s: %d runs, want 1", key, len(got[key]))
		}
		r := got[key][0]
		if r.status != status || r.fetched != fetched || r.new != new {
			t.Errorf("%s: %+v, want status=%s fetched=%d new=%d", key, r, status, fetched, new)
		}
		if (status == "failed") != (r.err != nil) {
			t.Errorf("%s: error = %v for status %s", key, r.err, status)
		}
	}
	check("fn_adhoc", "ok", 10, 10)
	check("fn_dax", "failed", 0, 0) // fetch error
	check("fn_atx", "ok", 0, 0)     // empty feed
	check("fn_usa", "failed", 0, 0) // parse error
	check("ots_finanzen", "ok", 15, 15)
	check("ots_wirtschaft", "ok", 0, 0)
	check("prn_all", "ok", 20, 20)
	check("voestalpine_ir", "ok", 10, 10)

	if n := count(t, pool, "SELECT count(*) FROM articles"); n != 55 {
		t.Errorf("%d articles, want 55", n)
	}
	// The Mercedes-Benz ad-hoc item is linked via its issuer.
	n := count(t, pool, `
		SELECT count(*) FROM article_entities e
		JOIN listings l ON l.company_id = e.company_id
		WHERE l.isin = 'DE0007100000' AND e.status = 'linked' AND e.method = 'issuer'`)
	if n != 1 {
		t.Errorf("mercedes links = %d, want 1", n)
	}
}

func TestRunDueLinksPressReleasesViaSender(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	// OTS names the sender in <author>: every Lenzing release is an issuer
	// link, including titles that only say "Lenzing Group"/"Lenzing Gruppe".
	n := count(t, pool, `
		SELECT count(*) FROM articles a
		JOIN sources s ON s.id = a.source_id
		JOIN article_entities e ON e.article_id = a.id
		JOIN companies c ON c.id = e.company_id
		WHERE s.key = 'ots_finanzen' AND c.slug = 'lenzing'
		  AND e.status = 'linked' AND e.method = 'issuer'`)
	if n != 9 {
		t.Errorf("lenzing issuer links from OTS = %d, want 9", n)
	}
}

func TestRunDueLinksIRFeedToItsIssuer(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Every item of voestalpine's own feed is about voestalpine, also when
	// the title does not name it.
	n := count(t, pool, `
		SELECT count(*) FROM articles a
		JOIN sources s ON s.id = a.source_id
		JOIN article_entities e ON e.article_id = a.id
		JOIN companies c ON c.id = e.company_id
		WHERE s.key = 'voestalpine_ir' AND c.slug = 'voestalpine'
		  AND e.status = 'linked' AND e.method = 'issuer'`)
	if n != 10 {
		t.Errorf("voestalpine issuer links = %d, want 10", n)
	}
	n = count(t, pool, `
		SELECT count(*) FROM articles a
		JOIN article_links l ON l.article_id = a.id
		JOIN companies c ON c.id = l.company_id
		WHERE a.title LIKE 'Next milestone for greentec steel%' AND c.slug = 'voestalpine'
		  AND l.status = 'linked'`)
	if n != 1 {
		t.Errorf("title without company name: %d links, want 1", n)
	}
}

func TestRunDueRespectsFetchInterval(t *testing.T) {
	r, pool, f, c := newRunner(t)
	ctx := context.Background()
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}

	c.now = c.now.Add(9 * time.Minute) // fetch_interval is 10 min, also after failures
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if f.calls[urlAdhoc] != 1 || f.calls[urlDAX] != 1 {
		t.Fatalf("refetched too early: %v", f.calls)
	}

	c.now = c.now.Add(2 * time.Minute)
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if f.calls[urlAdhoc] != 2 {
		t.Fatalf("adhoc calls = %d, want 2", f.calls[urlAdhoc])
	}
	adhoc := runs(t, pool)["fn_adhoc"]
	if len(adhoc) != 2 || adhoc[1].status != "ok" || adhoc[1].fetched != 10 || adhoc[1].new != 0 {
		t.Fatalf("second adhoc run = %+v, want ok fetched=10 new=0", adhoc)
	}
	if n := count(t, pool, "SELECT count(*) FROM articles"); n != 55 {
		t.Errorf("%d articles after refetch, want 55", n)
	}
}

func TestRunDueSkipsDisabledAndUnknownParser(t *testing.T) {
	r, pool, f, _ := newRunner(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE sources SET enabled = false WHERE key = 'fn_dax'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE sources SET parser = 'nope' WHERE key = 'fn_atx'`); err != nil {
		t.Fatal(err)
	}
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}

	got := runs(t, pool)
	if len(got["fn_dax"]) != 0 || f.calls[urlDAX] != 0 {
		t.Errorf("disabled source ran: %+v", got["fn_dax"])
	}
	if len(got["fn_atx"]) != 1 || got["fn_atx"][0].status != "skipped" || f.calls[urlATX] != 0 {
		t.Errorf("unknown parser: runs=%+v calls=%d, want one skipped run and no fetch", got["fn_atx"], f.calls[urlATX])
	}
}

func TestRunDueRecoversFromParserPanic(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	for name := range r.Parsers {
		r.Parsers[name] = func([]byte) ([]ingest.RawArticle, error) { panic("boom") }
	}
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	for key, rs := range runs(t, pool) {
		if key == "fn_dax" {
			continue // fails at fetch, before the parser
		}
		if len(rs) != 1 || rs[0].status != "failed" {
			t.Errorf("%s: %+v, want one failed run", key, rs)
		}
	}
}

type fakeTDM struct {
	reserved map[string]bool
	errs     map[string]error
	calls    map[string]int
}

func (f *fakeTDM) CheckTDM(_ context.Context, url string) (bool, error) {
	f.calls[url]++
	if err := f.errs[url]; err != nil {
		return false, err
	}
	return f.reserved[url], nil
}

type tdmRow struct {
	reserved  bool
	checkedAt *time.Time
}

func tdmState(t *testing.T, pool *pgxpool.Pool, key string) tdmRow {
	t.Helper()
	var r tdmRow
	if err := pool.QueryRow(context.Background(),
		`SELECT tdm_reserved, tdm_checked_at FROM sources WHERE key = $1`, key).Scan(&r.reserved, &r.checkedAt); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestRunDueChecksTDMAtMostDaily(t *testing.T) {
	r, pool, _, c := newRunner(t)
	ctx := context.Background()
	tdm := &fakeTDM{reserved: map[string]bool{urlAdhoc: true}, calls: map[string]int{}}
	r.TDM = tdm
	start := c.now

	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if got := tdmState(t, pool, "fn_adhoc"); !got.reserved || got.checkedAt == nil || !got.checkedAt.Equal(start) {
		t.Errorf("fn_adhoc after check = %+v, want reserved at %v", got, start)
	}
	if got := tdmState(t, pool, "fn_atx"); got.reserved || got.checkedAt == nil {
		t.Errorf("fn_atx after check = %+v, want not reserved, checked", got)
	}
	// The check is independent of the feed: a failing fetch still gets checked.
	if tdm.calls[urlDAX] != 1 {
		t.Errorf("fn_dax checks = %d, want 1", tdm.calls[urlDAX])
	}

	c.now = start.Add(11 * time.Minute) // due again, but checked recently
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if tdm.calls[urlAdhoc] != 1 {
		t.Errorf("rechecked within 24h: %d calls", tdm.calls[urlAdhoc])
	}

	tdm.reserved[urlAdhoc] = false
	c.now = start.Add(24*time.Hour + time.Minute)
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if tdm.calls[urlAdhoc] != 2 {
		t.Errorf("adhoc checks after 24h = %d, want 2", tdm.calls[urlAdhoc])
	}
	if got := tdmState(t, pool, "fn_adhoc"); got.reserved || !got.checkedAt.Equal(c.now) {
		t.Errorf("fn_adhoc after recheck = %+v, want lifted at %v", got, c.now)
	}
}

func TestRunDueKeepsTDMStateWhenCheckFails(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `UPDATE sources SET tdm_reserved = true WHERE key = 'fn_adhoc'`); err != nil {
		t.Fatal(err)
	}
	r.TDM = &fakeTDM{errs: map[string]error{urlAdhoc: errors.New("timeout")}, calls: map[string]int{}}

	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	if got := tdmState(t, pool, "fn_adhoc"); !got.reserved || got.checkedAt != nil {
		t.Errorf("fn_adhoc = %+v, want reservation kept and unchecked", got)
	}
	// A failed check does not fail the ingest run.
	if rs := runs(t, pool)["fn_adhoc"]; len(rs) != 1 || rs[0].status != "ok" {
		t.Errorf("fn_adhoc runs = %+v, want one ok run", rs)
	}
}
