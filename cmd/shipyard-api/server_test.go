package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthAndReadyEndpoints(t *testing.T) {
	mux := newMux(discardLogger())

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/healthz"},
		{http.MethodGet, "/readyz"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s %s: status = %d, want %d", tc.method, tc.path, rec.Code, http.StatusOK)
			}
			if rec.Header().Get("X-Request-Id") == "" {
				t.Errorf("%s %s: missing X-Request-Id header", tc.method, tc.path)
			}
		})
	}
}

func TestHealthz_WrongMethodNotAllowed(t *testing.T) {
	mux := newMux(discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Go 1.22+'s ServeMux matches "GET /healthz" only for GET; any other
	// method for a registered path falls through to its default 405.
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

// TestGracefulShutdown_WaitsForInFlightRequest is the test the Phase 02
// acceptance criteria calls for: it proves that a slow, in-flight request
// completes successfully even though shutdown is triggered while it is
// still running — the whole point of using srv.Shutdown instead of just
// killing the process.
func TestGracefulShutdown_WaitsForInFlightRequest(t *testing.T) {
	const handlerDelay = 150 * time.Millisecond

	handlerStarted := make(chan struct{})
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(handlerStarted)
			time.Sleep(handlerDelay)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("done"))
		}),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open listener: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	type result struct {
		status int
		body   string
		err    error
	}
	reqDone := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String())
		if err != nil {
			reqDone <- result{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, _ := io.ReadAll(resp.Body)
		reqDone <- result{status: resp.StatusCode, body: string(body)}
	}()

	// Wait until the handler has actually started before shutting down,
	// so this test exercises "shutdown during an in-flight request" and
	// not "shutdown before the request even arrived".
	<-handlerStarted

	shutdownStart := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() returned error: %v", err)
	}
	shutdownElapsed := time.Since(shutdownStart)

	res := <-reqDone
	if res.err != nil {
		t.Fatalf("in-flight request failed: %v", res.err)
	}
	if res.status != http.StatusOK {
		t.Errorf("status = %d, want %d", res.status, http.StatusOK)
	}
	if res.body != "done" {
		t.Errorf("body = %q, want %q", res.body, "done")
	}
	// Shutdown must have actually waited for the handler, not returned
	// immediately and let the connection get cut off mid-response.
	if shutdownElapsed < handlerDelay {
		t.Errorf("Shutdown() returned after %s, want at least %s (should wait for in-flight handler)", shutdownElapsed, handlerDelay)
	}

	if err := <-serveErr; err != nil && err != http.ErrServerClosed {
		t.Errorf("Serve() returned unexpected error: %v", err)
	}
}
