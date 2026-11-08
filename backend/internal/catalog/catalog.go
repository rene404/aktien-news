package catalog

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/rene404/aktien-news/backend/internal/store/db"
)

var ErrNotFound = errors.New("catalog: not found")

// Catalog looks up master data.
type Catalog struct {
	q *db.Queries
}

func New(conn db.DBTX) *Catalog {
	return &Catalog{q: db.New(conn)}
}

func (c *Catalog) CompanyByISIN(ctx context.Context, isin string) (db.Company, error) {
	return notFound(c.q.CompanyByISIN(ctx, &isin))
}

func (c *Catalog) CompanyByTicker(ctx context.Context, mic, ticker string) (db.Company, error) {
	return notFound(c.q.CompanyByTicker(ctx, db.CompanyByTickerParams{Mic: mic, Ticker: ticker}))
}

func notFound[T any](v T, err error) (T, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

// ValidLEI checks the ISO 17442 format (18 upper-case alphanumerics, two
// check digits) and the check digits: the number with A=10 … Z=35 is 1 mod 97.
func ValidLEI(lei string) bool {
	if len(lei) != 20 {
		return false
	}
	rem := 0
	for i, r := range lei {
		switch {
		case r >= '0' && r <= '9':
			rem = (rem*10 + int(r-'0')) % 97
		case r >= 'A' && r <= 'Z' && i < 18:
			rem = (rem*100 + int(r-'A') + 10) % 97
		default:
			return false
		}
	}
	return rem == 1
}

// ValidISIN checks format and the ISO 6166 check digit (Luhn over the
// digit expansion where A=10 … Z=35).
func ValidISIN(isin string) bool {
	if len(isin) != 12 {
		return false
	}
	var digits []int
	for i, r := range isin {
		switch {
		case i < 2 && r >= 'A' && r <= 'Z':
		case i >= 2 && i < 11 && (r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'):
		case i == 11 && r >= '0' && r <= '9':
		default:
			return false
		}
		if r >= 'A' && r <= 'Z' {
			v := int(r-'A') + 10
			digits = append(digits, v/10, v%10)
		} else {
			digits = append(digits, int(r-'0'))
		}
	}
	sum := 0
	for i := len(digits) - 1; i >= 0; i-- {
		d := digits[i]
		if (len(digits)-1-i)%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}
