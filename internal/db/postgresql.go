package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/models"
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
	DB, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("goose: failed to open DB: %v", err)
	}

	err = goose.Up(DB, config.MigrationDir)
	if err != nil {
		return nil, fmt.Errorf("goose: failed to migrate DB: %v", err)
	}

	return &postgresStore{store: DB}, nil
}

// Save saves a new URL record to the database.
// If a URL record already exists, ErrConflict is returned.
func (pg *postgresStore) Save(ctx context.Context, u *models.URL) error {
	const q = `
		INSERT INTO url
			(id, short_url, original_url, user_id)
		VALUES
			($1, $2, $3, $4)
	`

	// query the database to insert the URL record
	_, err := pg.store.ExecContext(ctx, q, u.ID, u.ShortURL, u.OriginalURL, u.UserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// return ErrConflict if the record already exists
			if pgErr.Code == pgerrcode.UniqueViolation {
				return models.ErrConflict
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
func (pg *postgresStore) SaveAll(ctx context.Context, urls []*models.URL) error {
	const q = `
        INSERT INTO url 
            (id, short_url, original_url, user_id)
        VALUES
            ($1, $2, $3, $4)
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
		_, err := stmt.ExecContext(ctx, url.ID, url.ShortURL, url.OriginalURL, url.UserID)
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
func (pg *postgresStore) Get(ctx context.Context, sURL models.ShortURL) (*models.URL, error) {
	const q = `
		SELECT
			id, short_url, original_url
		FROM
			url
		WHERE
			short_url = $1
	`

	u := new(models.URL)
	err := pg.store.QueryRowContext(ctx, q, sURL).Scan(&u.ID, &u.ShortURL, &u.OriginalURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, models.ErrNotFound
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

func (pg *postgresStore) GetAllByUserID(ctx context.Context, userID string) ([]*models.URL, error) {
	const q = `
		SELECT
			short_url, original_url
		FROM
			url
		WHERE
			user_id = $1
	`
	rows, err := pg.store.QueryContext(ctx, q, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// Create a new error with additional context.
			return nil, fmt.Errorf("retrieve url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}

	all := make([]*models.URL, 0)
	for rows.Next() {
		u := new(models.URL)
		err = rows.Scan(&u.ShortURL, &u.OriginalURL)
		if err != nil {
			return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
		}
		all = append(all, u)
	}

	if err = rows.Close(); err != nil {
		return nil, fmt.Errorf("close rows with query (%s): %w", formatQuery(q), err)
	}

	// Rows.Err will report the last error encountered by Rows.Scan.
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}

	if len(all) == 0 {
		return nil, models.ErrNotFound
	}

	return all, nil
}

func (pg *postgresStore) DeleteURLs(ctx context.Context, urls ...*models.URL) error {
	const q = `
		UPDATE url
		SET is_deleted = TRUE
		WHERE user_id = $1
		AND short_url = $2
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
		_, err := stmt.ExecContext(ctx, url.UserID, url.ShortURL)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				// continue if the record already exists
				if pgErr.Code == pgerrcode.UniqueViolation {
					continue
				}
				// create a new error with additional context
				return fmt.Errorf("delete url with query (%s): %w",
					formatQuery(q), formatPgError(pgErr),
				)
			}

			return fmt.Errorf("delete url with query (%s): %w", formatQuery(q), err)
		}
	}

	return tx.Commit()
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
