// Package config loads and validates Shipyard's runtime configuration.
//
// Configuration comes from environment variables only (no config files, no
// flags) — this keeps every binary trivially runnable in a container, where
// environment variables are the standard way to configure a process.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds the control-plane API's runtime settings.
type Config struct {
	// Addr is the address the HTTP server listens on, e.g. ":8080".
	Addr string

	// ShutdownTimeout bounds how long the server waits for in-flight
	// requests to finish during a graceful shutdown before giving up.
	ShutdownTimeout time.Duration

	// DatabaseURL is a PostgreSQL connection string. The default points
	// at the local docker-compose stack (deployments/compose) so running
	// locally needs no environment setup; any real environment must
	// override it explicitly.
	DatabaseURL string
}

// Load reads configuration from the environment and validates it.
//
// It fails fast: an invalid or missing required value is reported here,
// at startup, rather than surfacing later as a confusing runtime error.
func Load() (Config, error) {
	cfg := Config{
		Addr:            getEnv("SHIPYARD_API_ADDR", ":8080"),
		ShutdownTimeout: 10 * time.Second,
		DatabaseURL:     getEnv("SHIPYARD_DATABASE_URL", "postgres://shipyard:shipyard@localhost:5433/shipyard?sslmode=disable"),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.Addr == "" {
		return fmt.Errorf("SHIPYARD_API_ADDR must not be empty")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive, got %s", c.ShutdownTimeout)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("SHIPYARD_DATABASE_URL must not be empty")
	}
	return nil
}

// getEnv returns the environment variable named key, or fallback if it is
// unset or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
