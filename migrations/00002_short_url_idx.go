package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upShortURLIdx, downShortURLIdx)
}

func upShortURLIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS short_url ON url (short_url)`)
	if err != nil {
		return fmt.Errorf("create short_url index: %w", err)
	}

	return nil
}

func downShortURLIdx(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS short_url`)
	if err != nil {
		return fmt.Errorf("drop short_url index: %w", err)
	}

	return nil
}
