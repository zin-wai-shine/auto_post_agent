package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"github.com/zinwaishine/super-agent/internal/database"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database migrations",
	Long: `Apply all pending database migrations to your PostgreSQL database.

This will create the required tables, indexes, and extensions
including pgvector for vector similarity search.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dbURL, _ := cmd.Flags().GetString("db-url")

		// Try to load from config if no URL provided
		if dbURL == "" {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("no --db-url provided and could not load config: %w", err)
			}
			dbURL = cfg.Database.URL
		}

		if dbURL == "" {
			return fmt.Errorf("database URL is required: use --db-url or run 'super-agent init' first")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fmt.Println("🗄️  Connecting to database...")
		db, err := database.New(ctx, config.DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: "5m",
		})
		if err != nil {
			return fmt.Errorf("database connection failed: %w", err)
		}
		defer db.Close()

		fmt.Println("🔄 Running migrations...")
		if err := db.Migrate(ctx); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}

		fmt.Println("✅ All migrations applied successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	migrateCmd.Flags().String("db-url", "", "PostgreSQL connection URL (overrides config)")
}
