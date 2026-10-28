package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TxBeginner is satisfied by *pgxpool.Pool (new transaction) and pgx.Tx
// (savepoint). Functions that open their own transaction take this, so a
// caller can still run them inside an outer transaction.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}
