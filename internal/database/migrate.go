package database

import (
	"context"
	_ "embed"
	"fmt"
)

//go:embed migrations/001_initial_schema.sql
var migrationInitialSchema string

// Migrate runs all database migrations in order.
func (db *DB) Migrate(ctx context.Context) error {
	migrations := []struct {
		name string
		sql  string
	}{
		{"001_initial_schema", migrationInitialSchema},
	}

	// Create migrations tracking table
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _migrations (
			id          SERIAL PRIMARY KEY,
			name        TEXT UNIQUE NOT NULL,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	for _, m := range migrations {
		// Check if already applied
		var exists bool
		err := db.Pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM _migrations WHERE name = $1)", m.name,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", m.name, err)
		}
		if exists {
			fmt.Printf("  ✓ Migration %s already applied\n", m.name)
			continue
		}

		// Apply migration
		_, err = db.Pool.Exec(ctx, m.sql)
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", m.name, err)
		}

		// Record migration
		_, err = db.Pool.Exec(ctx,
			"INSERT INTO _migrations (name) VALUES ($1)", m.name,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", m.name, err)
		}

		fmt.Printf("  ✅ Migration %s applied successfully\n", m.name)
	}

	return nil
}
