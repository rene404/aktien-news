package ingest_test

import (
	"context"
	"errors"
	"fmt"
	"html"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/catalog"
	"github.com/rene404/aktien-news/backend/internal/ingest"
	"github.com/rene404/aktien-news/backend/internal/ingest/finanznachrichten"
	"github.com/rene404/aktien-news/backend/internal/ingest/pressrss"
	"github.com/rene404/aktien-news/backend/internal/ingest/sec"
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
	urlSEC   = "https://www.sec.gov/cgi-bin/browse-edgar?action=getcurrent&type=8-K&company=&dateb=&owner=include&start=0&count=100&output=atom"
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
			urlSEC:   readFixture(t, "sec/testdata/current_8k.xml"),
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
			"sec":               sec.Parse,
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
	check("sec_8k", "ok", 3, 3)

	if n := count(t, pool, "SELECT count(*) FROM articles"); n != 58 {
		t.Errorf("%d articles, want 58", n)
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

func TestRunDueLinksFilingsByCIK(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	links := func(ext string) []string {
		t.Helper()
		rows, err := pool.Query(context.Background(), `
			SELECT c.slug || ':' || e.method || ':' || e.status
			FROM articles a JOIN article_entities e ON e.article_id = a.id JOIN companies c ON c.id = e.company_id
			WHERE a.external_id = $1 ORDER BY 1`, ext)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var l string
			if err := rows.Scan(&l); err != nil {
				t.Fatal(err)
			}
			out = append(out, l)
		}
		return out
	}
	if got := links("0000320193-26-000080"); !slices.Equal(got, []string{"apple:issuer:linked"}) {
		t.Errorf("apple 8-K links = %v", got)
	}
	if got := links("0001999999-26-000002"); len(got) != 0 {
		t.Errorf("unknown CIK links = %v, want none", got)
	}
}

func TestRunDueSkipsSourcesWithAReason(t *testing.T) {
	r, pool, f, _ := newRunner(t)
	tdm := &fakeTDM{calls: map[string]int{}}
	r.TDM = tdm
	r.Skip = map[string]string{"sec": "SEC_USER_AGENT not set"}
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	rs := runs(t, pool)["sec_8k"]
	if len(rs) != 1 || rs[0].status != "skipped" || rs[0].err == nil || *rs[0].err != "SEC_USER_AGENT not set" {
		t.Errorf("sec_8k runs = %+v, want one skipped run with the reason", rs)
	}
	if f.calls[urlSEC] != 0 || tdm.calls[urlSEC] != 0 {
		t.Errorf("skipped source was fetched (%d) or TDM-checked (%d)", f.calls[urlSEC], tdm.calls[urlSEC])
	}
}

func TestRunDueUsesPerParserFetcher(t *testing.T) {
	r, pool, f, _ := newRunner(t)
	secFetcher := &fakeFetcher{bodies: map[string][]byte{urlSEC: f.bodies[urlSEC]}, calls: map[string]int{}}
	r.Fetchers = map[string]ingest.Fetcher{"sec": secFetcher}
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if secFetcher.calls[urlSEC] != 1 || f.calls[urlSEC] != 0 {
		t.Errorf("sec fetched by override %d times, by default %d times", secFetcher.calls[urlSEC], f.calls[urlSEC])
	}
	if rs := runs(t, pool)["sec_8k"]; len(rs) != 1 || rs[0].status != "ok" {
		t.Errorf("sec_8k runs = %+v", rs)
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
	if n := count(t, pool, "SELECT count(*) FROM articles"); n != 58 {
		t.Errorf("%d articles after refetch, want 58", n)
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

// A source never checked successfully counts as reserved while its check
// fails: unknown is not "no reservation".
func TestRunDueTreatsUncheckedSourceAsReservedWhenCheckFails(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	r.TDM = &fakeTDM{errs: map[string]error{urlAdhoc: errors.New("timeout")}, calls: map[string]int{}}
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := tdmState(t, pool, "fn_adhoc"); !got.reserved || got.checkedAt != nil {
		t.Errorf("fn_adhoc = %+v, want reserved and still unchecked", got)
	}
}

// A database error while recording one source's TDM state must not stop
// the other sources.
func TestRunDueContinuesWhenRecordingTDMFails(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	ctx := context.Background()
	r.TDM = &fakeTDM{calls: map[string]int{}}
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION fail_tdm() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'boom'; END $$;
		CREATE TRIGGER fail_tdm BEFORE UPDATE OF tdm_checked_at ON sources
		FOR EACH ROW WHEN (OLD.key = 'fn_adhoc') EXECUTE FUNCTION fail_tdm();`); err != nil {
		t.Fatal(err)
	}
	if err := r.RunDue(ctx); err != nil {
		t.Fatalf("RunDue: %v", err)
	}
	got := runs(t, pool)
	for _, key := range []string{"fn_adhoc", "fn_dax", "ots_finanzen", "prn_all"} {
		if len(got[key]) != 1 {
			t.Errorf("%s: %d runs, want 1", key, len(got[key]))
		}
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

type feedItem struct {
	title, link string
	published   time.Time
}

// rssFeed builds a minimal RSS 2.0 document both feed parsers accept.
func rssFeed(items ...feedItem) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>t</title>`)
	for _, it := range items {
		fmt.Fprintf(&b, "<item><title>%s</title><link>%s</link><pubDate>%s</pubDate></item>",
			html.EscapeString(it.title), html.EscapeString(it.link), it.published.Format(time.RFC1123Z))
	}
	b.WriteString(`</channel></rss>`)
	return []byte(b.String())
}

type clusterRow struct {
	size           int
	members        int
	representative string // source key of the representative article
}

// clusterOf returns the cluster of the article with the given link.
func clusterOf(t *testing.T, pool *pgxpool.Pool, link string) (int64, clusterRow) {
	t.Helper()
	var id int64
	var c clusterRow
	err := pool.QueryRow(context.Background(), `
		SELECT c.id, c.size,
		       (SELECT count(*) FROM articles m WHERE m.cluster_id = c.id),
		       (SELECT s.key FROM articles r JOIN sources s ON s.id = r.source_id WHERE r.id = c.representative_article_id)
		FROM articles a JOIN clusters c ON c.id = a.cluster_id
		WHERE a.url = $1`, link).Scan(&id, &c.size, &c.members, &c.representative)
	if err != nil {
		t.Fatalf("cluster of %s: %v", link, err)
	}
	return id, c
}

func TestRunDueClustersTheSameReleaseAcrossSources(t *testing.T) {
	r, pool, f, c := newRunner(t)
	const title = "EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen"
	at := c.now.Add(-2 * time.Hour)
	// OTS publishes first; the ad-hoc source still represents the cluster.
	f.bodies[urlOTSF] = rssFeed(feedItem{title, "https://www.ots.at/presseaussendung/EQS_1/andritz", at})
	f.bodies[urlAdhoc] = rssFeed(feedItem{title, "https://www.finanznachrichten.de/nachrichten-2026-10/1-andritz.htm", at.Add(time.Minute)})
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}

	otsCluster, got := clusterOf(t, pool, "https://www.ots.at/presseaussendung/EQS_1/andritz")
	fnCluster, _ := clusterOf(t, pool, "https://www.finanznachrichten.de/nachrichten-2026-10/1-andritz.htm")
	if otsCluster != fnCluster {
		t.Fatalf("same release in clusters %d and %d", otsCluster, fnCluster)
	}
	if want := (clusterRow{size: 2, members: 2, representative: "fn_adhoc"}); got != want {
		t.Errorf("cluster = %+v, want %+v", got, want)
	}
}

func TestRunDueKeepsFollowUpsAndOldStoriesApart(t *testing.T) {
	r, pool, f, c := newRunner(t)
	const (
		corr23 = "EQS-PVR: KORREKTUR vom 23.10.2026: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 2 BörseG mit dem Ziel der europaweiten Verbreitung"
		corr27 = "EQS-PVR: KORREKTUR vom 27.10.2026: Lenzing AG: Veröffentlichung gemäß § 135 Abs. 2 BörseG mit dem Ziel der europaweiten Verbreitung"
		cms    = "EQS-CMS: Andritz AG: Sonstige Emittenten-/Unternehmensinformationen"
	)
	f.bodies[urlOTSF] = rssFeed(
		feedItem{corr23, "https://www.ots.at/a/23", c.now.Add(-time.Hour)},
		feedItem{corr27, "https://www.ots.at/a/27", c.now.Add(-time.Hour)},
		feedItem{cms, "https://www.ots.at/a/cms-old", c.now.Add(-100 * time.Hour)},
		feedItem{cms, "https://www.ots.at/a/cms-new", c.now.Add(-time.Hour)},
	)
	if err := r.RunDue(context.Background()); err != nil {
		t.Fatal(err)
	}

	for _, pair := range [][2]string{
		{"https://www.ots.at/a/23", "https://www.ots.at/a/27"},
		{"https://www.ots.at/a/cms-old", "https://www.ots.at/a/cms-new"}, // 99 h apart
	} {
		a, ca := clusterOf(t, pool, pair[0])
		b, cb := clusterOf(t, pool, pair[1])
		if a == b || ca.size != 1 || cb.size != 1 {
			t.Errorf("%s and %s: clusters %d/%+v and %d/%+v, want two singletons", pair[0], pair[1], a, ca, b, cb)
		}
	}
}

func TestClusterUnassignedBackfillsOldArticles(t *testing.T) {
	r, pool, _, _ := newRunner(t)
	ctx := context.Background()
	if err := r.RunDue(ctx); err != nil {
		t.Fatal(err)
	}
	// Articles from before clustering existed: no fingerprint, no cluster.
	if _, err := pool.Exec(ctx, `UPDATE articles SET cluster_id = NULL, simhash = NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM clusters`); err != nil {
		t.Fatal(err)
	}

	if err := r.ClusterUnassigned(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, pool, `SELECT count(*) FROM articles WHERE cluster_id IS NULL OR simhash IS NULL`); n != 0 {
		t.Errorf("%d articles still unclustered", n)
	}
	if n := count(t, pool, `SELECT count(*) FROM clusters c WHERE size <> (SELECT count(*) FROM articles a WHERE a.cluster_id = c.id)`); n != 0 {
		t.Errorf("%d clusters with a wrong size", n)
	}
}
