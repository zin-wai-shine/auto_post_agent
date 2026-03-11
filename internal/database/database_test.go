package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/zinwaishine/super-agent/internal/config"
)

// getTestDBURL returns the database URL for integration tests.
// Set SUPERAGENT_TEST_DB_URL env var, or falls back to local Docker default.
func getTestDBURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("SUPERAGENT_TEST_DB_URL")
	if url == "" {
		url = "postgres://superagent:superagent_dev_2024@localhost:5433/superagent?sslmode=disable"
	}
	return url
}

func TestNew_Connect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := New(ctx, config.DatabaseConfig{
		URL:             getTestDBURL(t),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
	})
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if db.Pool == nil {
		t.Fatal("pool is nil after successful connection")
	}
}

func TestHealth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := New(ctx, config.DatabaseConfig{
		URL:             getTestDBURL(t),
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: "5m",
	})
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer db.Close()

	if err := db.Health(ctx); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

func TestNew_InvalidURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := New(ctx, config.DatabaseConfig{
		URL:          "postgres://invalid:invalid@localhost:59999/nope?sslmode=disable",
		MaxOpenConns: 2,
		MaxIdleConns: 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid connection, got nil")
	}
}

func TestClose_NilPool(t *testing.T) {
	db := &DB{Pool: nil}
	// Should not panic
	db.Close()
}
