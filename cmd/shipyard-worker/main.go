// Command shipyard-worker is the execution-plane worker: it claims jobs
// from the durable queue and runs them.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ISHANT57/shipyard/internal/config"
	"github.com/ISHANT57/shipyard/internal/queue"
	"github.com/ISHANT57/shipyard/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.LoadWorker()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("worker exited with error", "error", err)
		os.Exit(1)
	}
}

// run wires the worker and blocks until it has drained and stopped.
func run(cfg config.WorkerConfig, logger *slog.Logger) error {
	// SIGINT/SIGTERM cancel ctx, which starts a graceful drain rather than
	// killing in-flight jobs (see internal/worker).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("creating database pool: %w", err)
	}
	defer pool.Close()
	// Fail fast on a bad connection string, like the API does.
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	q := queue.New(pool, queue.Options{Lease: cfg.Lease})

	w := worker.New(q, placeholderHandler(logger), worker.Config{
		ID:           cfg.ID,
		Concurrency:  cfg.Concurrency,
		PollInterval: cfg.PollInterval,
		ReapInterval: cfg.ReapInterval,
		DrainTimeout: cfg.DrainTimeout,
	}, logger)

	return w.Run(ctx)
}

// placeholderHandler stands in for real stage execution. Phase 06
// replaces it with the sandbox runner (throwaway containers); until then
// it only logs, so the queue, leases, retries and shutdown behavior can be
// exercised end to end. It is idempotent trivially: it does nothing.
func placeholderHandler(logger *slog.Logger) worker.Handler {
	return func(ctx context.Context, job queue.Job) error {
		logger.Info("placeholder handler ran", "job", job.ID, "stage", job.StageID)
		return nil
	}
}
