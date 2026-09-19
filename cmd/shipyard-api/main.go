// Command shipyard-api is the control-plane HTTP server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ISHANT57/shipyard/internal/config"
	"github.com/ISHANT57/shipyard/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	if err := run(cfg, logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

// run wires the server and blocks until it shuts down cleanly or fails.
// Separated from main() so it returns an error instead of calling
// os.Exit itself — that keeps it testable.
func run(cfg config.Config, logger *slog.Logger) error {
	// signal.NotifyContext turns SIGINT/SIGTERM into context cancellation:
	// ctx.Done() closes the moment the process receives either signal,
	// instead of the process dying immediately.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	s, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer s.Close()

	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: newMux(logger, s),
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		// ListenAndServe always returns a non-nil error; ErrServerClosed
		// means Shutdown was called deliberately, which is not a failure.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining in-flight requests")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Shutdown stops accepting new connections immediately, but waits for
	// active requests to finish (or for shutdownCtx to expire) before
	// returning — this is the graceful part.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("shutdown complete")
	return nil
}
