package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zinwaishine/super-agent/internal/config"
	"github.com/zinwaishine/super-agent/internal/database"
	"github.com/zinwaishine/super-agent/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server for the React dashboard",
	Long: `Launch a lightweight Go API server that the React Frontend
queries via Vector Similarity Search.

Endpoints:
  GET  /api/health          — Health check
  GET  /api/listings        — List all listings (paginated)
  GET  /api/listings/:id    — Get a single listing
  POST /api/search          — Natural language vector search
  GET  /api/pipeline        — Pipeline status overview`,
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		host, _ := cmd.Flags().GetString("host")
		dbURL, _ := cmd.Flags().GetString("db-url")

		// Try to load DB URL from config if not provided
		if dbURL == "" {
			cfg, err := config.Load()
			if err == nil && cfg.Database.URL != "" {
				dbURL = cfg.Database.URL
			}
		}
		if dbURL == "" {
			return fmt.Errorf("database URL is required: use --db-url or run 'super-agent init' first")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Connect to database
		db, err := database.New(ctx, config.DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: "5m",
		})
		if err != nil {
			return fmt.Errorf("database connection failed: %w", err)
		}
		defer db.Close()

		// Create and start server
		srv := server.New(db.Pool, host, port)

		// Graceful shutdown
		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			fmt.Println("\n⏳ Shutting down gracefully...")
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			srv.Shutdown(shutdownCtx)
		}()

		return srv.Start()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	serveCmd.Flags().String("host", "0.0.0.0", "host to bind to")
	serveCmd.Flags().String("db-url", "", "PostgreSQL connection URL (overrides config)")
	serveCmd.Flags().Bool("cors", true, "enable CORS for frontend dev")
}
