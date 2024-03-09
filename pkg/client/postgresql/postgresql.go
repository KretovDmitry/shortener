package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/KretovDmitry/shortener/pkg/retries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
}

func NewClient(ctx context.Context, maxAttempts int, dsn string) (pool *pgxpool.Pool, err error) {
	err = retries.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if pool, err = pgxpool.New(ctx, dsn); err != nil {
			return fmt.Errorf("unable to create connection pool: %v", err)
		}

		return nil
	}, maxAttempts, 5*time.Second)

	if err != nil {
		return nil, fmt.Errorf("new postgresql client: %v", err)
	}

	return
}
