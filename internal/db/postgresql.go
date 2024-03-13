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

	// create the short_url_idx index if it does not exist
	if _, err := tx.Exec(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS short_url ON url (short_url)
		`); err != nil {
		return fmt.Errorf("create short_url index: %w", err)
	}

	return tx.Commit(ctx)
}

// Save saves a new URL record to the database.
// If a URL record already exists, the record is not inserted.
func (pg *postgresStore) Save(ctx context.Context, u *URL) error {
	q := `
        INSERT INTO url
            (short_url, original_url)
        VALUES
            ($1, $2)
        RETURNING id
    `

	// FIXME: r.ID remains empty if the URL record already exists.
	// Query the database to insert the URL record.
	if err := pg.store.QueryRow(ctx, q, u.ShortURL, u.OriginalURL).Scan(&u.ID); err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// Check if the error is a unique constraint violation.
			// Constraint violation means that the URL record already exists.
			// All fields are unique, so we can safely ignore this error.
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

// SaveAll saves multiple URL records to the database in a single transaction.
// If a URL record already exists, the record is not inserted.
// If the transaction fails, all changes are rolled back.
func (pg *postgresStore) SaveAll(ctx context.Context, u []*URL) error {
	tx, err := pg.store.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO url 
            (short_url, original_url)
        VALUES
            ($1, $2)
    `

	for _, url := range u {
		if _, err := tx.Exec(ctx, query, url.ShortURL, url.OriginalURL); err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok {
				// Check if the error is a unique constraint violation.
				// Constraint violation means that the URL record already exists.
				// All fields are unique, so we can safely ignore this error.
				if pgErr.Code == "23505" {
					continue
				}
				// Create a new error with additional context.
				return fmt.Errorf("save url with query (%s): %w",
					formatQuery(query), formatPgError(pgErr),
				)
			}

			return fmt.Errorf("save url with query (%s): %w", formatQuery(query), err)
		}
	}

	return tx.Commit(ctx)
}

// Get retrieves a URL record from the database based on its short URL.
// If the URL record does not exist, this function returns the ErrURLNotFound error.
func (pg *postgresStore) Get(ctx context.Context, sURL ShortURL) (*URL, error) {
	q := `
		SELECT
			id, short_url, original_url
		FROM
			url
		WHERE
			short_url = $1
		`

	record := new(URL)
	err := pg.store.QueryRow(ctx, q, sURL).Scan(&record.ID, &record.ShortURL, &record.OriginalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrURLNotFound
		}
		if pgErr, ok := err.(*pgconn.PgError); ok {
			// Create a new error with additional context.
			return nil, fmt.Errorf("retrieve url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}

	return record, nil
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
