package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upDeletedCol, downDeletedCol)
}

func upDeletedCol(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE IF EXISTS url 
		ADD COLUMN IF NOT EXISTS is_deleted boolean
		DEFAULT	FALSE
	`)
	if err != nil {
		return fmt.Errorf(
			"url table: add is_deleted column: %w", err,
		)
	}

	return nil
}

func downDeletedCol(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		ALTER TABLE IF EXISTS url 
		DROP COLUMN IF EXISTS is_deleted
	`)
	if err != nil {
		return fmt.Errorf(
			"url table: drop is_deleted column: %w", err,
		)
	}

	return nil
}
