package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upUserIDIdx, downUserIDIdx)
}

func upUserIDIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS user_id ON url (user_id)`)
	if err != nil {
		return fmt.Errorf("create user_id index: %w", err)
	}

	return nil
}

func downUserIDIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS user_id`)
	if err != nil {
		return fmt.Errorf("drop user_id index: %w", err)
	}

	return nil
}
