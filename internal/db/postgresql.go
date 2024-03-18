package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KretovDmitry/shortener/pkg/retries"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	store *pgxpool.Pool
}

// NewPostgresStore creates a new Postgres database connection pool
// and initializes the database schema.
func NewPostgresStore(ctx context.Context, dsn string) (*postgresStore, error) {
	var (
		maxAttempts = 3
		pool        *pgxpool.Pool
		err         error
	)

	// try to connect to the database several times
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

	// initialize the database schema
	if err = bootstrap(ctx, pool); err != nil {
		return nil, fmt.Errorf("bootstrap: %w", err)
	}

	return &postgresStore{
		store: pool,
	}, nil
}

// bootstrap initializes the database schema.
func bootstrap(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// create the "url" table if it does not exist
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS public.url (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			short_url VARCHAR(8) NOT NULL UNIQUE,
			original_url VARCHAR(255) NOT NULL UNIQUE
		);
	`); err != nil {
		return fmt.Errorf("create url table: %w", err)
	}

	// create the short_url index if it does not exist
	if _, err := tx.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS short_url ON url (short_url)
		`); err != nil {
		return fmt.Errorf("create short_url index: %w", err)
	}

	// create the original_url index if it does not exist
	if _, err := tx.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS original_url ON url (original_url)
		`); err != nil {
		return fmt.Errorf("create original_url index: %w", err)
	}

	return tx.Commit(ctx)
}

// Save saves a new URL record to the database.
// If a URL record already exists, ErrConflict is returned.
func (pg *postgresStore) Save(ctx context.Context, u *URL) error {
	const q = `
        INSERT INTO url
            (short_url, original_url)
        VALUES
            ($1, $2)
    `

	// query the database to insert the URL record
	if _, err := pg.store.Exec(ctx, q, u.ShortURL, u.OriginalURL); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// return ErrConflict if the record already exists
			if pgErr.Code == pgerrcode.UniqueViolation {
				return ErrConflict
			}
			// create a new error with additional context
			return fmt.Errorf("save url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return fmt.Errorf("save url with query (%s): %w", formatQuery(q), err)
	}

	return nil
}

// SaveAll saves multiple URL records to the database in a single transaction.
// If a URL record already exists, the record is not inserted.
// If the transaction fails, all changes are rolled back.
func (pg *postgresStore) SaveAll(ctx context.Context, u []*URL) error {
	tx, err := pg.store.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const q = `
        INSERT INTO url 
            (short_url, original_url)
        VALUES
            ($1, $2)
    `

	for _, url := range u {
		if _, err := tx.Exec(ctx, q, url.ShortURL, url.OriginalURL); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				// continue if the record already exists
				if pgErr.Code == pgerrcode.UniqueViolation {
					continue
				}
				// create a new error with additional context
				return fmt.Errorf("save url with query (%s): %w",
					formatQuery(q), formatPgError(pgErr),
				)
			}

			return fmt.Errorf("save url with query (%s): %w", formatQuery(q), err)
		}
	}

	return tx.Commit(ctx)
}

// Get retrieves a URL record from the database based on its short URL.
// If the URL record does not exist, ErrURLNotFound is returned.
func (pg *postgresStore) Get(ctx context.Context, sURL ShortURL) (*URL, error) {
	const q = `
		SELECT
			id, short_url, original_url
		FROM
			url
		WHERE
			short_url = $1
		`

	u := new(URL)
	if err := pg.store.QueryRow(ctx, q, sURL).Scan(&u.ID, &u.ShortURL, &u.OriginalURL); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrURLNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Create a new error with additional context.
			return nil, fmt.Errorf("retrieve url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}

	return u, nil
}

// Ping verifies the connection to the database is alive.
func (pg *postgresStore) Ping(ctx context.Context) error {
	return pg.store.Ping(ctx)
}

// formatQuery removes tabs and replaces newlines with spaces in the given query string.
func formatQuery(q string) string {
	return strings.ReplaceAll(strings.ReplaceAll(q, "\t", ""), "\n", " ")
}

// formatPgError formats a PgError into a human-friendly error message.
func formatPgError(err *pgconn.PgError) error {
	return fmt.Errorf("SQL Error: %s, Detail: %s, Where: %s, Code: %s, SQLState: %s",
		err.Message,
		err.Detail,
		err.Where,
		err.Code,
		err.SQLState(),
	)
}
