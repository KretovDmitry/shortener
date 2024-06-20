package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type URLRepository struct {
	db     *sql.DB
	logger logger.Logger
}

// NewPostgresStore creates a new Postgres database connection pool
// and initializes the database schema.
func NewURLRepository(
	ctx context.Context,
	db *sql.DB,
	logger logger.Logger,
) (*URLRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: *sql.DB", errs.ErrNilDependency)
	}
	if logger == nil {
		return nil, fmt.Errorf("%w: logger", errs.ErrNilDependency)
	}
	return &URLRepository{db: db, logger: logger}, nil
}

// Save saves a new URL record to the database.
// If a URL record already exists, ErrConflict is returned.
func (ur *URLRepository) Save(ctx context.Context, u *models.URL) error {
	const q = `
		INSERT INTO url
			(id, short_url, original_url, user_id)
		VALUES
			($1, $2, $3, $4)
	`

	// query the database to insert the URL record
	_, err := ur.db.ExecContext(ctx, q, u.ID, u.ShortURL, u.OriginalURL, u.UserID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// return ErrConflict if the record already exists
			if pgErr.Code == pgerrcode.UniqueViolation {
				return errs.ErrConflict
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
func (ur *URLRepository) SaveAll(ctx context.Context, urls []*models.URL) error {
	const q = `
        INSERT INTO url 
            (id, short_url, original_url, user_id)
        VALUES
            ($1, $2, $3, $4)
    `

	tx, err := ur.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil {
			if errors.Is(err, sql.ErrTxDone) {
				ur.logger.Errorf("rollback: %v", err)
			}
		}
	}()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err = stmt.Close(); err != nil {
			if errors.Is(err, sql.ErrTxDone) {
				ur.logger.Errorf("close prepared statement: %v", err)
			}
		}
	}()

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
func (ur *URLRepository) Get(ctx context.Context, sURL models.ShortURL) (*models.URL, error) {
	const q = `
		SELECT
			id, short_url, original_url, is_deleted
		FROM
			url
		WHERE
			short_url = $1
	`

	u := new(models.URL)
	err := ur.db.QueryRowContext(ctx, q, sURL).Scan(
		&u.ID,
		&u.ShortURL,
		&u.OriginalURL,
		&u.IsDeleted,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errs.ErrNotFound
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

// GetAllByUserID retrieves all URL records from the database associated with a specific user.
// It returns a slice of URL pointers and an error if any occurred.
// If no URL records are found for the given user, it returns nil and ErrNotFound.
func (ur *URLRepository) GetAllByUserID(ctx context.Context, userID string) ([]*models.URL, error) {
	const q = `
        SELECT
            short_url, original_url
        FROM
            url
        WHERE
            user_id = $1
    `

	// Execute the query with the given userID.
	rows, err := ur.db.QueryContext(ctx, q, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// If the error is a PgError, create a new error with additional context.
			return nil, fmt.Errorf("retrieve url with query (%s): %w",
				formatQuery(q), formatPgError(pgErr),
			)
		}

		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}
	// Close the rows when the function returns.
	defer func() {
		if err = rows.Close(); err != nil {
			ur.logger.Errorf("close rows: %v", err)
		}
	}()

	all := make([]*models.URL, 0) // Initialize an empty slice to store the URL pointers.
	for rows.Next() {
		u := new(models.URL) // Create a new URL pointer.

		// Scan the current row into the URL pointer.
		err = rows.Scan(&u.ShortURL, &u.OriginalURL)
		if err != nil {
			return nil, fmt.Errorf(
				"retrieve url with query (%s): %w", formatQuery(q), err,
			)
		}

		// Append the URL pointer to the slice.
		all = append(all, u)
	}

	// Check if there was an error during iteration over the rows.
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("retrieve url with query (%s): %w", formatQuery(q), err)
	}

	// If the slice is empty, return nil and ErrNotFound.
	if len(all) == 0 {
		return nil, errs.ErrNotFound
	}

	// Return the slice of URL pointers and nil error.
	return all, nil
}

// DeleteURLs deletes the specified URLs from the database.
// It takes a context and a slice of URL pointers as parameters.
// It returns an error if any occurs during the deletion process.
// If no URLs are provided, it returns nil.
func (ur *URLRepository) DeleteURLs(ctx context.Context, urls ...*models.URL) error {
	if len(urls) == 0 {
		return nil
	}

	const q = "UPDATE url SET is_deleted = TRUE WHERE short_url = $1;"

	tx, err := ur.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err = tx.Rollback(); err != nil {
			if errors.Is(err, sql.ErrTxDone) {
				ur.logger.Errorf("rollback: %v", err)
			}
		}
	}()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer func() {
		if err = stmt.Close(); err != nil {
			if errors.Is(err, sql.ErrTxDone) {
				ur.logger.Errorf("close prepared statement: %v", err)
			}
		}
	}()

	for _, url := range urls {
		_, err := stmt.ExecContext(ctx, url.ShortURL)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) {
				return fmt.Errorf("delete url with query (%s): %w",
					formatQuery(q), formatPgError(pgErr),
				)
			}
			return fmt.Errorf("delete url with query (%s): %w",
				formatQuery(q), err)
		}
	}

	return tx.Commit()
}

// Ping verifies the connection to the database is alive.
func (ur *URLRepository) Ping(ctx context.Context) error {
	return ur.db.PingContext(ctx)
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
