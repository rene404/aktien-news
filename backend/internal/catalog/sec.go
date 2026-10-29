package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

// SECCompany is one issuer from SEC's company_tickers_exchange.json with its
// NYSE/Nasdaq listings.
type SECCompany struct {
	CIK      string // decimal, no leading zeros (as in companies.cik)
	Name     string
	Listings []SECListing
}

type SECListing struct {
	Ticker string // class shares with a dot, as in the seed: BRK.B
	MIC    string
}

var secExchangeMIC = map[string]string{"NYSE": "XNYS", "Nasdaq": "XNAS"}

// ParseSECTickers reads company_tickers_exchange.json
// ({"fields":["cik","name","ticker","exchange"],"data":[[...],...]}). It keeps
// NYSE and Nasdaq rows and groups them by CIK in file order.
func ParseSECTickers(b []byte) ([]SECCompany, error) {
	var doc struct {
		Fields []string            `json:"fields"`
		Data   [][]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("sec tickers: %w", err)
	}
	if !slices.Equal(doc.Fields, []string{"cik", "name", "ticker", "exchange"}) {
		return nil, fmt.Errorf("sec tickers: unexpected fields %v", doc.Fields)
	}

	var out []SECCompany
	index := map[string]int{}
	for i, row := range doc.Data {
		if len(row) != 4 {
			return nil, fmt.Errorf("sec tickers: row %d has %d columns", i, len(row))
		}
		var cik json.Number
		dec := json.NewDecoder(bytes.NewReader(row[0]))
		dec.UseNumber()
		if err := dec.Decode(&cik); err != nil {
			return nil, fmt.Errorf("sec tickers: row %d: cik: %w", i, err)
		}
		if _, err := cik.Int64(); err != nil {
			return nil, fmt.Errorf("sec tickers: row %d: cik %q: %w", i, cik, err)
		}
		var name, ticker, exchange *string
		for j, dst := range []**string{&name, &ticker, &exchange} {
			if err := json.Unmarshal(row[j+1], dst); err != nil {
				return nil, fmt.Errorf("sec tickers: row %d column %d: %w", i, j+1, err)
			}
		}
		if name == nil || ticker == nil || exchange == nil || *ticker == "" {
			continue
		}
		mic, ok := secExchangeMIC[*exchange]
		if !ok {
			continue
		}
		key := strings.TrimLeft(cik.String(), "0")
		l := SECListing{Ticker: strings.ReplaceAll(*ticker, "-", "."), MIC: mic}
		if k, ok := index[key]; ok {
			out[k].Listings = append(out[k].Listings, l)
			continue
		}
		index[key] = len(out)
		out = append(out, SECCompany{CIK: key, Name: *name, Listings: []SECListing{l}})
	}
	return out, nil
}

// ImportSEC merges the SEC list into the catalog in one transaction. Running
// it again changes nothing.
//
//   - A curated company found by CIK, or else by one of the tickers, keeps its
//     name, aliases and listings; only a missing CIK is filled in.
//   - An uncurated company found by CIK gets missing listings.
//   - Any other issuer becomes a new uncurated company (slug sec-<cik>).
//
// Listings already owned by another company are left alone.
func ImportSEC(ctx context.Context, pool *pgxpool.Pool, companies []SECCompany) error {
	var created, filled int
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		for _, c := range companies {
			cik := c.CIK
			id, curated, err := companyForSEC(ctx, q, c)
			if err != nil {
				return fmt.Errorf("cik %s: %w", cik, err)
			}
			switch {
			case id != 0 && curated:
				if err := q.FillCompanyCIK(ctx, db.FillCompanyCIKParams{ID: id, Cik: &cik}); err != nil {
					return fmt.Errorf("cik %s: %w", cik, err)
				}
				filled++
				continue
			case id == 0:
				id, err = q.InsertUncuratedCompany(ctx, db.InsertUncuratedCompanyParams{Slug: "sec-" + cik, Name: c.Name, Cik: &cik})
				if err != nil {
					return fmt.Errorf("cik %s: %w", cik, err)
				}
				created++
			}
			for _, l := range c.Listings {
				if err := q.InsertListingIfAbsent(ctx, db.InsertListingIfAbsentParams{CompanyID: id, Ticker: l.Ticker, Mic: l.MIC}); err != nil {
					return fmt.Errorf("cik %s listing %s:%s: %w", cik, l.MIC, l.Ticker, err)
				}
			}
		}
		return nil
	})
	if err == nil {
		slog.Info("sec import done", "issuers", len(companies), "created", created, "curated_matched", filled)
	}
	return err
}

// companyForSEC finds the company an SEC issuer belongs to: by CIK, else a
// curated company owning one of its tickers. 0 means none.
func companyForSEC(ctx context.Context, q *db.Queries, c SECCompany) (int64, bool, error) {
	byCIK, err := q.CompanyByCIK(ctx, &c.CIK)
	if err == nil {
		return byCIK.ID, byCIK.Curated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	for _, l := range c.Listings {
		owner, err := q.CompanyByListing(ctx, db.CompanyByListingParams{Mic: l.MIC, Ticker: l.Ticker})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, false, err
		}
		if owner.Curated && owner.Cik == nil {
			return owner.ID, true, nil
		}
	}
	return 0, false, nil
}
