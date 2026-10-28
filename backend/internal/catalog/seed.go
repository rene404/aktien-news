// Package catalog owns master data: sources, companies, listings, index
// memberships and aliases. Seed loads it from CSV; Catalog looks it up.
package catalog

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

var (
	knownMICs     = []string{"XWBO", "XETR", "XNAS", "XNYS"}
	knownMarkets  = []string{"AT", "DE", "US", "MULTI"}
	knownKinds    = []string{"adhoc", "press_release", "news_api", "ir", "filing"}
	knownPolicies = []string{"none", "feed_fields", "full_text"}
	aliasKinds    = []string{"core_name", "former_name", "brand", "issuer_name"}
)

// Seed validates all CSV files in dir and upserts them in one transaction.
// Nothing is written if any row is invalid. Running it twice is a no-op.
func Seed(ctx context.Context, pool *pgxpool.Pool, dir fs.FS) error {
	data, err := load(dir)
	if err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		return data.write(ctx, db.New(tx))
	})
}

type seedData struct {
	sources     []db.UpsertSourceParams
	companies   []db.UpsertCompanyParams
	listings    []listingRow
	memberships []db.UpsertIndexMembershipParams
	aliases     []db.UpsertAliasParams
}

type listingRow struct {
	companySlug string
	params      db.UpsertListingParams
}

func load(dir fs.FS) (*seedData, error) {
	var d seedData
	var errs []error
	addErr := func(file string, line int, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s:%d: %s", file, line, fmt.Sprintf(format, args...)))
	}

	err := readCSV(dir, "sources.csv", []string{"key", "name", "kind", "market", "url", "weight", "fetch_interval_minutes", "text_policy", "terms_url"},
		func(line int, r map[string]string) {
			if !slices.Contains(knownKinds, r["kind"]) {
				addErr("sources.csv", line, "unknown kind %q", r["kind"])
			}
			if !slices.Contains(knownMarkets, r["market"]) {
				addErr("sources.csv", line, "unknown market %q", r["market"])
			}
			if !slices.Contains(knownPolicies, r["text_policy"]) {
				addErr("sources.csv", line, "unknown text_policy %q", r["text_policy"])
			}
			var weight pgtype.Numeric
			if w, err := strconv.ParseFloat(r["weight"], 64); err != nil || w < 0 || w > 1 {
				addErr("sources.csv", line, "weight %q must be a number in [0,1]", r["weight"])
			} else if err := weight.Scan(r["weight"]); err != nil {
				addErr("sources.csv", line, "weight: %v", err)
			}
			minutes, err := strconv.Atoi(r["fetch_interval_minutes"])
			if err != nil || minutes < 1 {
				addErr("sources.csv", line, "fetch_interval_minutes %q must be a positive integer", r["fetch_interval_minutes"])
			}
			policy := db.TextPolicy(r["text_policy"])
			d.sources = append(d.sources, db.UpsertSourceParams{
				Key: r["key"], Name: r["name"], Kind: db.SourceKind(r["kind"]), Market: r["market"], Url: r["url"],
				Weight:        weight,
				FetchInterval: pgtype.Interval{Microseconds: int64(minutes) * int64(time.Minute/time.Microsecond), Valid: true},
				TextPolicy:    &policy,
				TermsUrl:      optional(r["terms_url"]),
			})
		})
	errs = append(errs, err)

	slugs := map[string]bool{}
	err = readCSV(dir, "companies.csv", []string{"slug", "name", "legal_name", "country", "lei", "cik"},
		func(line int, r map[string]string) {
			if r["slug"] == "" || slugs[r["slug"]] {
				addErr("companies.csv", line, "missing or duplicate slug %q", r["slug"])
			}
			slugs[r["slug"]] = true
			if len(r["country"]) != 2 || strings.ToUpper(r["country"]) != r["country"] {
				addErr("companies.csv", line, "country %q must be ISO alpha-2", r["country"])
			}
			d.companies = append(d.companies, db.UpsertCompanyParams{
				Slug: r["slug"], Name: r["name"], LegalName: optional(r["legal_name"]),
				Country: r["country"], Lei: optional(r["lei"]), Cik: optional(r["cik"]),
			})
		})
	errs = append(errs, err)

	listingKeys := map[string]bool{}
	err = readCSV(dir, "listings.csv", []string{"company_slug", "isin", "ticker", "mic", "currency"},
		func(line int, r map[string]string) {
			if !slugs[r["company_slug"]] {
				addErr("listings.csv", line, "unknown company_slug %q", r["company_slug"])
			}
			if r["isin"] != "" && !ValidISIN(r["isin"]) {
				addErr("listings.csv", line, "invalid ISIN %s", r["isin"])
			}
			if !slices.Contains(knownMICs, r["mic"]) {
				addErr("listings.csv", line, "unknown mic %q", r["mic"])
			}
			key := r["mic"] + ":" + r["ticker"]
			if r["ticker"] == "" || listingKeys[key] {
				addErr("listings.csv", line, "missing or duplicate listing %s", key)
			}
			listingKeys[key] = true
			d.listings = append(d.listings, listingRow{
				companySlug: r["company_slug"],
				params: db.UpsertListingParams{
					Isin: optional(r["isin"]), Ticker: r["ticker"], Mic: r["mic"], Currency: optional(r["currency"]),
				},
			})
		})
	errs = append(errs, err)

	err = readCSV(dir, "index_memberships.csv", []string{"mic", "ticker", "index_key", "valid_from"},
		func(line int, r map[string]string) {
			if !listingKeys[r["mic"]+":"+r["ticker"]] {
				addErr("index_memberships.csv", line, "unknown listing %s:%s", r["mic"], r["ticker"])
			}
			from, err := time.Parse(time.DateOnly, r["valid_from"])
			if err != nil {
				addErr("index_memberships.csv", line, "valid_from %q is not YYYY-MM-DD", r["valid_from"])
			}
			d.memberships = append(d.memberships, db.UpsertIndexMembershipParams{
				IndexKey: r["index_key"], ValidFrom: pgtype.Date{Time: from, Valid: true}, Mic: r["mic"], Ticker: r["ticker"],
			})
		})
	errs = append(errs, err)

	err = readCSV(dir, "company_aliases.csv", []string{"company_slug", "alias", "kind"},
		func(line int, r map[string]string) {
			if !slugs[r["company_slug"]] {
				addErr("company_aliases.csv", line, "unknown company_slug %q", r["company_slug"])
			}
			if !slices.Contains(aliasKinds, r["kind"]) {
				addErr("company_aliases.csv", line, "unknown kind %q", r["kind"])
			}
			d.aliases = append(d.aliases, db.UpsertAliasParams{Alias: r["alias"], Kind: r["kind"], CompanySlug: r["company_slug"]})
		})
	errs = append(errs, err)

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &d, nil
}

func (d *seedData) write(ctx context.Context, q *db.Queries) error {
	for _, s := range d.sources {
		if _, err := q.UpsertSource(ctx, s); err != nil {
			return fmt.Errorf("source %s: %w", s.Key, err)
		}
	}
	companyIDs := map[string]int64{}
	for _, c := range d.companies {
		id, err := q.UpsertCompany(ctx, c)
		if err != nil {
			return fmt.Errorf("company %s: %w", c.Slug, err)
		}
		companyIDs[c.Slug] = id
	}
	for _, l := range d.listings {
		l.params.CompanyID = companyIDs[l.companySlug]
		if _, err := q.UpsertListing(ctx, l.params); err != nil {
			return fmt.Errorf("listing %s:%s: %w", l.params.Mic, l.params.Ticker, err)
		}
	}
	for _, m := range d.memberships {
		if err := q.UpsertIndexMembership(ctx, m); err != nil {
			return fmt.Errorf("membership %s %s:%s: %w", m.IndexKey, m.Mic, m.Ticker, err)
		}
	}
	for _, a := range d.aliases {
		if err := q.UpsertAlias(ctx, a); err != nil {
			return fmt.Errorf("alias %s/%s: %w", a.CompanySlug, a.Alias, err)
		}
	}
	return nil
}

// readCSV checks the header and calls fn once per data row (1-based line
// numbers including the header, as an editor shows them).
func readCSV(dir fs.FS, name string, header []string, fn func(line int, row map[string]string)) error {
	f, err := dir.Open(name)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	got, err := r.Read()
	if err != nil {
		return fmt.Errorf("%s: read header: %w", name, err)
	}
	if !slices.Equal(got, header) {
		return fmt.Errorf("%s: header %v, want %v", name, got, header)
	}
	for line := 2; ; line++ {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s:%d: %w", name, line, err)
		}
		row := make(map[string]string, len(header))
		for i, col := range header {
			row[col] = strings.TrimSpace(rec[i])
		}
		fn(line, row)
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
