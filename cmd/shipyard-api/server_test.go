package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ISHANT57/shipyard/internal/store"
	"github.com/ISHANT57/shipyard/internal/testdb"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestStoreAndMux gives each test its own real, migrated database
// (via internal/testdb) and the actual production mux wired against it —
// these tests exercise real wiring (config -> store -> handlers), not a
// stand-in, which is exactly the part of cmd/shipyard-api a plain unit
// test on handlers.go alone would not cover.
func newTestStoreAndMux(t *testing.T) (*store.Store, http.Handler) {
	t.Helper()
	dsn := testdb.NewPostgres(t)

	s, err := store.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	t.Cleanup(s.Close)

	return s, newMux(discardLogger(), s)
}

func TestHealthAndReadyEndpoints(t *testing.T) {
	_, mux := newTestStoreAndMux(t)

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
	_, mux := newTestStoreAndMux(t)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Go 1.22+'s ServeMux matches "GET /healthz" only for GET; any other
	// method for a registered path falls through to its default 405.
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /healthz: status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCreateProjectAndPipeline_EndToEnd(t *testing.T) {
	_, mux := newTestStoreAndMux(t)

	// Create a project through the real HTTP handler, not store directly
	// — this exercises JSON decoding, validation, and encoding together.
	projBody, _ := json.Marshal(createProjectRequest{Name: "shipyard", RepoURL: "https://github.com/ISHANT57/shipyard"})
	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(projBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /projects: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var proj projectResponse
	if err := json.NewDecoder(rec.Body).Decode(&proj); err != nil {
		t.Fatalf("decoding project response: %v", err)
	}
	if proj.ID == "" {
		t.Fatal("project response has empty ID")
	}

	// Submit a pipeline for it, twice, with the same idempotency key.
	pipeBody, _ := json.Marshal(createPipelineRequest{ProjectID: proj.ID, IdempotencyKey: "e2e-key"})

	req1 := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(pipeBody))
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first POST /pipelines: status = %d, body = %s", rec1.Code, rec1.Body.String())
	}
	var first pipelineResponse
	_ = json.NewDecoder(rec1.Body).Decode(&first)

	req2 := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(pipeBody))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	// Second submission with the same key: same pipeline, but 200 (not
	// created), not 201 — the status code itself proves the endpoint
	// distinguishes "created" from "already existed".
	if rec2.Code != http.StatusOK {
		t.Fatalf("second POST /pipelines: status = %d, want %d, body = %s", rec2.Code, http.StatusOK, rec2.Body.String())
	}
	var second pipelineResponse
	_ = json.NewDecoder(rec2.Body).Decode(&second)

	if second.ID != first.ID {
		t.Errorf("second submission returned a different pipeline: got %s, want %s", second.ID, first.ID)
	}
}

func TestCreatePipeline_UnknownProjectReturns404(t *testing.T) {
	_, mux := newTestStoreAndMux(t)

	body, _ := json.Marshal(createPipelineRequest{ProjectID: "00000000-0000-0000-0000-000000000000", IdempotencyKey: "some-key"})
	req := httptest.NewRequest(http.MethodPost, "/pipelines", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d, body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestCreateProject_MissingFieldsReturns400(t *testing.T) {
	_, mux := newTestStoreAndMux(t)

	body, _ := json.Marshal(createProjectRequest{Name: "", RepoURL: ""})
	req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
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
