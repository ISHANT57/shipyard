// Package config loads and validates Shipyard's runtime configuration.
//
// Configuration comes from environment variables only (no config files, no
// flags) — this keeps every binary trivially runnable in a container, where
// environment variables are the standard way to configure a process.
package config

import (
	"fmt"
	"os"
	"strconv"
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

// WorkerConfig holds the execution-plane worker's runtime settings.
type WorkerConfig struct {
	// DatabaseURL is the PostgreSQL connection string (same default and
	// override rules as Config.DatabaseURL).
	DatabaseURL string

	// ID identifies this worker process in job leases and logs. Each
	// concurrent slot appends its own index, so the fencing check can tell
	// slots apart.
	ID string

	// Concurrency is how many jobs this process runs at once.
	Concurrency int

	// PollInterval is how often an idle slot re-checks the queue when no
	// LISTEN/NOTIFY wake-up arrives (ADR-009: polling is the safety net).
	PollInterval time.Duration

	// Lease is how long a claim stays valid without a heartbeat.
	Lease time.Duration

	// ReapInterval is how often this worker looks for expired leases.
	ReapInterval time.Duration

	// DrainTimeout bounds how long shutdown waits for in-flight jobs.
	DrainTimeout time.Duration
}

// LoadWorker reads the worker's configuration from the environment and
// validates it, failing fast like Load.
func LoadWorker() (WorkerConfig, error) {
	cfg := WorkerConfig{
		DatabaseURL: getEnv("SHIPYARD_DATABASE_URL", "postgres://shipyard:shipyard@localhost:5433/shipyard?sslmode=disable"),
		ID:          getEnv("SHIPYARD_WORKER_ID", defaultWorkerID()),
	}

	var err error
	if cfg.Concurrency, err = getEnvInt("SHIPYARD_WORKER_CONCURRENCY", 4); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}
	if cfg.PollInterval, err = getEnvDuration("SHIPYARD_WORKER_POLL_INTERVAL", 2*time.Second); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}
	if cfg.Lease, err = getEnvDuration("SHIPYARD_WORKER_LEASE", 30*time.Second); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}
	if cfg.ReapInterval, err = getEnvDuration("SHIPYARD_WORKER_REAP_INTERVAL", 10*time.Second); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}
	if cfg.DrainTimeout, err = getEnvDuration("SHIPYARD_WORKER_DRAIN_TIMEOUT", 30*time.Second); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return WorkerConfig{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

func (c WorkerConfig) validate() error {
	switch {
	case c.DatabaseURL == "":
		return fmt.Errorf("SHIPYARD_DATABASE_URL must not be empty")
	case c.ID == "":
		return fmt.Errorf("SHIPYARD_WORKER_ID must not be empty")
	case c.Concurrency < 1:
		return fmt.Errorf("SHIPYARD_WORKER_CONCURRENCY must be at least 1, got %d", c.Concurrency)
	case c.PollInterval <= 0:
		return fmt.Errorf("SHIPYARD_WORKER_POLL_INTERVAL must be positive, got %s", c.PollInterval)
	case c.Lease <= 0:
		return fmt.Errorf("SHIPYARD_WORKER_LEASE must be positive, got %s", c.Lease)
	case c.ReapInterval <= 0:
		return fmt.Errorf("SHIPYARD_WORKER_REAP_INTERVAL must be positive, got %s", c.ReapInterval)
	case c.DrainTimeout <= 0:
		return fmt.Errorf("SHIPYARD_WORKER_DRAIN_TIMEOUT must be positive, got %s", c.DrainTimeout)
	}
	return nil
}

// defaultWorkerID is "<hostname>-<pid>", unique enough to tell two worker
// processes apart in logs and leases without any coordination.
func defaultWorkerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

// getEnvInt is getEnv for integers. A set-but-unparseable value is an
// error, not a silent fallback to the default: a typo in a config value
// should stop startup, not be quietly ignored.
func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not an integer", key, v)
	}
	return n, nil
}

// getEnvDuration is getEnv for durations such as "500ms" or "30s".
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %q is not a duration (examples: 500ms, 30s)", key, v)
	}
	return d, nil
}
