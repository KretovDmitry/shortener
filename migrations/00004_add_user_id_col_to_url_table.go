package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(
		upAddUserIDColumnToURLTable,
		downAddUserIDColumnToURLTable,
	)
}

func upAddUserIDColumnToURLTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE IF EXISTS public.url
		ADD COLUMN IF NOT EXISTS user_id UUID;
	`)
	if err != nil {
		return fmt.Errorf("add user_id column to URL table: %w", err)
	}

	return nil
}

func downAddUserIDColumnToURLTable(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE IF EXISTS public.url
        DROP COLUMN IF EXISTS user_id;
	`)
	if err != nil {
		return fmt.Errorf("drop user_id column in URL table: %w", err)
	}

	return nil
}
