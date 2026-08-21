package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// AssertSchemaCurrent refuses to start against a database that predates the
// uuid baseline.
//
// goose tracks versions by number, so a database left at version 21 sees the
// single baseline file, decides it is already applied, and reports "no
// migrations to run". Nothing is wrong from goose's point of view, but the old
// integer schema is still in place while the application sends uuids - so the
// server boots, serves users (that column was always uuid), and then dies on
// the first listing insert with "invalid input syntax for type integer".
//
// A half-working server is worse than one that will not start, and the fix is
// one command, so say so rather than letting it be rediscovered.
func AssertSchemaCurrent(ctx context.Context, db *sql.DB) error {
	const query = `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relname = 'listings'
		  AND a.attname = 'id'`

	var columnType string
	switch err := db.QueryRowContext(ctx, query).Scan(&columnType); {
	case errors.Is(err, sql.ErrNoRows):
		// No listings table at all: an empty database that has not been
		// migrated yet. That is migrate-up's problem to report, not ours.
		return nil
	case err != nil:
		return fmt.Errorf("checking the schema version: %w", err)
	}

	if columnType == "uuid" {
		return nil
	}

	return fmt.Errorf(
		"this database predates the uuid migration (listings.id is %s, not uuid).\n"+
			"goose reports \"no migrations to run\" because it only counts versions, so\n"+
			"migrate-up will not fix it. Recreate the database:\n\n"+
			"    docker compose down -v && docker compose up -d db\n"+
			"    cd backend && make migrate-up && make seed",
		columnType,
	)
}
