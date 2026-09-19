package main

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
)

// requestIDKey is an unexported type for the context key, following Go's
// convention: an unexported type prevents collisions with context keys
// set by other packages, which a plain string key would not.
type requestIDKey struct{}

// newMux builds the API's routes, wrapped in its middleware chain. It
// returns http.Handler, not *http.ServeMux, because the middleware wraps
// the mux in plain function values that only satisfy the interface —
// this is the "accept interfaces, return structs" idiom in reverse: the
// caller (main, or a test) only ever needs to call ServeHTTP, so the
// interface is the honest return type.
func newMux(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// Liveness: "is the process running and able to respond at all".
	// It never checks dependencies — a database outage should not make
	// the orchestrator think this process needs restarting.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Readiness: "is the process ready to accept real traffic". Today
	// this is identical to liveness because there are no dependencies
	// yet (Phase 03 adds PostgreSQL, at which point /readyz starts
	// checking the database connection and /healthz still won't).
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return withRequestID(withLogging(mux, logger))
}

// withRequestID assigns a short random ID to every request and stores it
// in the request's context, so later middleware/handlers (and eventually
// OpenTelemetry spans, in Phase 09) can tag their output with it.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := newRequestID()
		w.Header().Set("X-Request-Id", id)
		ctx := context_WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withLogging logs each request's method, path, and assigned request ID
// after it completes.
func withLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", requestIDFromContext(r.Context()),
		)
	})
}

func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
