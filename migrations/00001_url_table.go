package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upURL, downURL)
}

func upURL(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS public.url (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			short_url VARCHAR(255) NOT NULL,
			original_url TEXT NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("create url table: %w", err)
	}

	return nil
}

func downURL(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS public.url;`)
	if err != nil {
		return fmt.Errorf("drop url table: %w", err)
	}

	return nil
}
