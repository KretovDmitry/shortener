package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

type postgresStore struct {
	store *sql.DB
}

// NewPostgresStore creates a new Postgres database connection pool
// and initializes the database schema.
func NewPostgresStore(ctx context.Context, dsn string) (*postgresStore, error) {
	var (
		DB  *sql.DB
		err error
	)

	err = retry.Do(func() error {
		DB, err = goose.OpenDBWithDriver("pgx", dsn)
		if err != nil {
			return err
		}

		return nil
	},
		retry.Context(ctx),
		retry.Attempts(3),
		retry.Delay(1*time.Second),
	)

	if err != nil {
		return nil, fmt.Errorf("goose: failed to open DB: %v", err)
	}

	err = goose.Up(DB, ".")
	if err != nil {
		return nil, fmt.Errorf("goose: failed to migrate DB: %v", err)
	}

	return &postgresStore{store: DB}, nil
}

// Save saves a new URL record to the database.
// If a URL record already exists, ErrConflict is returned.
func (pg *postgresStore) Save(ctx context.Context, u *URL) error {
	const q = `
		INSERT INTO url
			(id, short_url, original_url)
		VALUES
			($1, $2, $3)
	`

	// query the database to insert the URL record
	_, err := pg.store.ExecContext(ctx, q, u.ID, u.ShortURL, u.OriginalURL)
	if err != nil {
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
func (pg *postgresStore) SaveAll(ctx context.Context, urls []*URL) error {
	const q = `
        INSERT INTO url 
            (id, short_url, original_url)
        VALUES
            ($1, $2, $3)
    `

	tx, err := pg.store.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}

	for _, url := range urls {
		_, err := stmt.ExecContext(ctx, url.ID, url.ShortURL, url.OriginalURL)
		if err != nil {
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

	return tx.Commit()
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
	err := pg.store.QueryRowContext(ctx, q, sURL).Scan(&u.ID, &u.ShortURL, &u.OriginalURL)
	if err != nil {
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
	return pg.store.PingContext(ctx)
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
