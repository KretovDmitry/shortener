package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upOriginalURLIdx, downOriginalURLIdx)
}

func upOriginalURLIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS original_url ON url (original_url)`)
	if err != nil {
		return fmt.Errorf("create original_url index: %w", err)
	}

	return nil
}

func downOriginalURLIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS original_url`)
	if err != nil {
		return fmt.Errorf("drop original_url index: %w", err)
	}

	return nil
}
