package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KretovDmitry/shortener/pkg/retries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	store *pgxpool.Pool
}

// NewPostgresStore creates a new Postgres database connection pool.
func NewPostgresStore(ctx context.Context, dsn string) (*postgresStore, error) {
	var (
		maxAttempts = 3
		pool        *pgxpool.Pool
		err         error
	)

	err = retries.Do(func() error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if pool, err = pgxpool.New(ctx, dsn); err != nil {
			return fmt.Errorf("unable to create connection pool: %w", err)
		}

		return nil
	}, maxAttempts, 5*time.Second)

	if err != nil {
		return nil, fmt.Errorf("new postgresql client: %w", err)
	}

	// Execute a query to create the URL table if it does not exist.
	result, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.url (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			short_url VARCHAR(8) NOT NULL UNIQUE,
			original_url VARCHAR(255) NOT NULL UNIQUE
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create table with status %s: %w", result, err)
	}

	return &postgresStore{
		store: pool,
	}, nil
}

// SaveURL saves a new URL record to the database.
// If the URL record already exists, this function returns nil.
func (pg *postgresStore) SaveURL(ctx context.Context, r *URLRecord) error {
	q := `
        INSERT INTO url
            (short_url, original_url)
        VALUES
            ($1, $2)
        RETURNING id
    `

	// Query the database to insert the URL record.
	if err := pg.store.QueryRow(ctx, q, r.ShortURL, r.OriginalURL).Scan(&r.ID); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// Check if the error is a unique constraint violation.
			if pgErr.Code == "23505" {
				return nil
			}
			// Create a new error with additional context.
			return fmt.Errorf("save url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return fmt.Errorf("save url with query (%s): %w", formatQuery(q), err)
	}

	return nil
}

// RetrieveInitialURL retrieves the original URL associated with a short URL.
// If the short URL does not exist, an error of type ErrURLNotFound is returned.
func (pg *postgresStore) RetrieveInitialURL(ctx context.Context, sURL ShortURL) (OriginalURL, error) {
	q := "SELECT original_url FROM url WHERE short_url = $1"

	var originalURL string
	if err := pg.store.QueryRow(ctx, q, sURL).Scan(&originalURL); err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrURLNotFound
		}
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// Create a new error with additional context.
			return "", fmt.Errorf("retrieve url with query (%s): %w",
				q, formatPgError(pgErr),
			)
		}

		return "", fmt.Errorf("retrieve url with query (%s): %w", q, err)
	}

	return OriginalURL(originalURL), nil
}

func (pg *postgresStore) Ping(ctx context.Context) error {
	return pg.store.Ping(ctx)
}

func formatQuery(q string) string {
	return strings.ReplaceAll(strings.ReplaceAll(q, "\t", ""), "\n", " ")
}

func formatPgError(err *pgconn.PgError) error {
	return fmt.Errorf(
		"SQL Error: %s, Detail: %s, Where: %s, Code: %s, SQLState: %s",
		err.Message,
		err.Detail,
		err.Where,
		err.Code,
		err.SQLState(),
	)
}
