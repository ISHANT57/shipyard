package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ISHANT57/shipyard/internal/queue"
	"github.com/ISHANT57/shipyard/internal/testdb"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// env is one test's private database plus a queue tuned for fast tests:
// tiny backoff so retries happen in milliseconds, and a short lease.
type env struct {
	q       *queue.Queue
	pool    *pgxpool.Pool
	stageID uuid.UUID
}

func newEnv(t *testing.T, lease time.Duration) env {
	t.Helper()
	dsn := testdb.NewPostgres(t)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	q := queue.New(pool, queue.Options{
		Lease:       lease,
		BackoffBase: 5 * time.Millisecond,
		BackoffMax:  20 * time.Millisecond,
	})
	return env{q: q, pool: pool, stageID: testdb.SeedStage(t, pool)}
}

func (e env) enqueue(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := e.q.Enqueue(context.Background(), e.stageID)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	return id
}

func (e env) status(t *testing.T, id uuid.UUID) (status string, attempt int) {
	t.Helper()
	if err := e.pool.QueryRow(context.Background(),
		`SELECT status, attempt FROM jobs WHERE id = $1`, id).Scan(&status, &attempt); err != nil {
		t.Fatalf("status(%s): %v", id, err)
	}
	return status, attempt
}

func (e env) countStatus(t *testing.T, status string) int {
	t.Helper()
	var n int
	if err := e.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM jobs WHERE status = $1`, status).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// waitFor polls cond until it is true or the deadline passes.
func waitFor(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatalf("timed out after %s waiting for: %s", timeout, what)
}

// running is a started Worker plus a way to stop it and collect Run's result.
type running struct {
	stop func() error
}

func start(t *testing.T, w *Worker) running {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()

	var once sync.Once
	var result error
	stop := func() error {
		once.Do(func() {
			cancel()
			select {
			case result = <-done:
			case <-time.After(15 * time.Second):
				t.Error("Run() did not return within 15s of cancellation")
			}
		})
		return result
	}
	t.Cleanup(func() { _ = stop() })
	return running{stop: stop}
}

func TestWorker_ProcessesAllJobs(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	const jobs = 25
	for i := 0; i < jobs; i++ {
		e.enqueue(t)
	}

	var handled atomic.Int64
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		handled.Add(1)
		return nil
	}, Config{ID: "t", Concurrency: 4, PollInterval: 20 * time.Millisecond, ReapInterval: time.Second}, discardLogger())
	r := start(t, w)

	waitFor(t, 20*time.Second, "all jobs succeeded", func() bool { return e.countStatus(t, "succeeded") == jobs })
	if err := r.stop(); err != nil {
		t.Errorf("Run() error = %v, want nil after a clean drain", err)
	}
	if got := handled.Load(); got != jobs {
		t.Errorf("handler ran %d times, want exactly %d", got, jobs)
	}
}

// TestWorker_NotifyWakesIdleWorker proves LISTEN/NOTIFY works. The poll
// interval is 30s, so a job enqueued while the worker idles can only be
// picked up within seconds if the notification woke it.
func TestWorker_NotifyWakesIdleWorker(t *testing.T) {
	e := newEnv(t, 5*time.Second)

	w := New(e.q, func(ctx context.Context, job queue.Job) error { return nil },
		Config{ID: "t", Concurrency: 2, PollInterval: 30 * time.Second, ReapInterval: time.Minute}, discardLogger())
	start(t, w)

	// Wait until the listener has really subscribed, so the test does not
	// enqueue before anyone is listening (that notification would be lost
	// by design; polling would cover it, but that is not what is tested).
	waitFor(t, 10*time.Second, "LISTEN subscription established", func() bool {
		var n int
		err := e.pool.QueryRow(context.Background(),
			`SELECT count(*) FROM pg_stat_activity WHERE query ILIKE 'LISTEN shipyard_jobs%'`).Scan(&n)
		return err == nil && n > 0
	})

	id := e.enqueue(t)
	waitFor(t, 5*time.Second, "job picked up via NOTIFY, not the 30s poll", func() bool {
		s, _ := e.status(t, id)
		return s == "succeeded"
	})
}

func TestWorker_FailedJobIsRetriedAndSucceeds(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	id := e.enqueue(t)

	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		if job.Attempt == 1 {
			return errors.New("transient failure")
		}
		return nil
	}, Config{ID: "t", Concurrency: 2, PollInterval: 20 * time.Millisecond, ReapInterval: time.Second}, discardLogger())
	start(t, w)

	waitFor(t, 10*time.Second, "job succeeded on retry", func() bool {
		s, _ := e.status(t, id)
		return s == "succeeded"
	})
	if _, attempt := e.status(t, id); attempt != 2 {
		t.Errorf("attempt = %d, want 2 (one failure, one success)", attempt)
	}
}

func TestWorker_PoisonJobEndsInDeadLetter(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	id := e.enqueue(t)
	if _, err := e.pool.Exec(context.Background(), `UPDATE jobs SET max_attempts = 3 WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int64
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		calls.Add(1)
		return errors.New("always fails")
	}, Config{ID: "t", Concurrency: 2, PollInterval: 20 * time.Millisecond, ReapInterval: time.Second}, discardLogger())
	start(t, w)

	waitFor(t, 10*time.Second, "job dead-lettered", func() bool {
		s, _ := e.status(t, id)
		return s == "dead"
	})
	// Give the pool a moment: a dead job must never be run again.
	time.Sleep(200 * time.Millisecond)
	if got := calls.Load(); got != 3 {
		t.Errorf("handler ran %d times, want exactly max_attempts = 3", got)
	}
}

func TestWorker_PanicIsContainedAndRetried(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	id := e.enqueue(t)

	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		if job.Attempt == 1 {
			panic("handler exploded")
		}
		return nil
	}, Config{ID: "t", Concurrency: 2, PollInterval: 20 * time.Millisecond, ReapInterval: time.Second}, discardLogger())
	start(t, w)

	waitFor(t, 10*time.Second, "job succeeded after a panicking first attempt", func() bool {
		s, _ := e.status(t, id)
		return s == "succeeded"
	})
	var msg *string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT error FROM job_attempts WHERE job_id = $1 AND attempt_number = 1`, id).Scan(&msg); err != nil {
		t.Fatal(err)
	}
	if msg == nil || *msg != "panic: handler exploded" {
		t.Errorf("attempt 1 error = %v, want \"panic: handler exploded\"", msg)
	}
}

// TestWorker_HeartbeatKeepsLongJobAlive: the job outlives its lease three
// times over while a reaper runs every 50ms. Only the heartbeat stands
// between the job and being reaped and run a second time.
func TestWorker_HeartbeatKeepsLongJobAlive(t *testing.T) {
	e := newEnv(t, 300*time.Millisecond)
	id := e.enqueue(t)

	var calls atomic.Int64
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		calls.Add(1)
		select {
		case <-time.After(1200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, Config{ID: "t", Concurrency: 2, PollInterval: 20 * time.Millisecond, ReapInterval: 50 * time.Millisecond}, discardLogger())
	start(t, w)

	waitFor(t, 15*time.Second, "long job succeeded", func() bool {
		s, _ := e.status(t, id)
		return s == "succeeded"
	})
	if s, attempt := e.status(t, id); s != "succeeded" || attempt != 1 {
		t.Errorf("status/attempt = %s/%d, want succeeded/1 (never reaped)", s, attempt)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("handler ran %d times, want exactly 1", got)
	}
}

// TestWorker_RecoversJobFromCrashedWorker: a "ghost" claims a job and
// never reports back — exactly what a crashed process looks like from the
// database. A real worker's reaper reclaims it and finishes the job.
func TestWorker_RecoversJobFromCrashedWorker(t *testing.T) {
	e := newEnv(t, 200*time.Millisecond)
	id := e.enqueue(t)

	if _, err := e.q.Claim(context.Background(), "ghost/0"); err != nil {
		t.Fatalf("ghost Claim() error = %v", err)
	}

	var calls atomic.Int64
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		calls.Add(1)
		return nil
	}, Config{ID: "rescuer", Concurrency: 2, PollInterval: 20 * time.Millisecond, ReapInterval: 50 * time.Millisecond}, discardLogger())
	start(t, w)

	waitFor(t, 15*time.Second, "crashed worker's job completed by a live worker", func() bool {
		s, _ := e.status(t, id)
		return s == "succeeded"
	})
	if _, attempt := e.status(t, id); attempt != 2 {
		t.Errorf("attempt = %d, want 2 (ghost's lost attempt + rescuer's)", attempt)
	}
	var msg *string
	if err := e.pool.QueryRow(context.Background(),
		`SELECT error FROM job_attempts WHERE job_id = $1 AND attempt_number = 1`, id).Scan(&msg); err != nil {
		t.Fatal(err)
	}
	if msg == nil || *msg != "lease expired" {
		t.Errorf("ghost attempt error = %v, want \"lease expired\"", msg)
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("handler ran %d times, want 1 (the ghost never ran it)", got)
	}
}

// TestWorker_GracefulDrain: shutdown arrives mid-job. The in-flight job
// must finish and be recorded; a job that was still waiting must not be
// started.
func TestWorker_GracefulDrain(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	first := e.enqueue(t)

	started := make(chan struct{})
	var once sync.Once
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		once.Do(func() { close(started) })
		// A well-behaved handler stops when its context is cancelled. If
		// shutdown wrongly cancelled it, this would return an error and the
		// job would fail instead of finishing: that is what this test
		// guards against.
		select {
		case <-time.After(400 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}, Config{ID: "t", Concurrency: 1, PollInterval: 20 * time.Millisecond, ReapInterval: time.Minute, DrainTimeout: 5 * time.Second}, discardLogger())
	r := start(t, w)

	<-started
	second := e.enqueue(t) // arrives while the only slot is busy

	begin := time.Now()
	if err := r.stop(); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if elapsed := time.Since(begin); elapsed < 200*time.Millisecond {
		t.Errorf("Run() returned after %s: it did not wait for the in-flight job", elapsed)
	}

	if s, _ := e.status(t, first); s != "succeeded" {
		t.Errorf("in-flight job status = %q, want succeeded (drained, not dropped)", s)
	}
	if s, attempt := e.status(t, second); s != "queued" || attempt != 0 {
		t.Errorf("waiting job = %s/attempt %d, want queued/0 (not started during shutdown)", s, attempt)
	}
}

// TestWorker_DrainTimeoutAbandonsStuckJob: a handler that ignores its
// context cannot hold shutdown hostage forever.
func TestWorker_DrainTimeoutCancelsInFlightJob(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	id := e.enqueue(t)

	started := make(chan struct{})
	var once sync.Once
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		once.Do(func() { close(started) })
		<-ctx.Done() // only ends when the drain timeout cancels it
		return ctx.Err()
	}, Config{ID: "t", Concurrency: 1, PollInterval: 20 * time.Millisecond, ReapInterval: time.Minute, DrainTimeout: 300 * time.Millisecond}, discardLogger())
	r := start(t, w)

	<-started
	begin := time.Now()
	if err := r.stop(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(begin); elapsed > 5*time.Second {
		t.Errorf("Run() took %s, want it bounded by the 300ms drain timeout", elapsed)
	}
	// The abandoned attempt is recorded as failed and queued for retry.
	if s, attempt := e.status(t, id); s != "queued" || attempt != 1 {
		t.Errorf("status/attempt = %s/%d, want queued/1 (failed attempt, will retry)", s, attempt)
	}
}

func TestWorker_ConcurrencyIsBounded(t *testing.T) {
	e := newEnv(t, 5*time.Second)
	const (
		limit = 3
		jobs  = 12
	)
	for i := 0; i < jobs; i++ {
		e.enqueue(t)
	}

	var inFlight, peak atomic.Int64
	w := New(e.q, func(ctx context.Context, job queue.Job) error {
		n := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(80 * time.Millisecond)
		return nil
	}, Config{ID: "t", Concurrency: limit, PollInterval: 20 * time.Millisecond, ReapInterval: time.Second}, discardLogger())
	start(t, w)

	waitFor(t, 20*time.Second, "all jobs done", func() bool { return e.countStatus(t, "succeeded") == jobs })
	if got := peak.Load(); got != limit {
		t.Errorf("peak concurrency = %d, want exactly %d (never above the limit, and actually using it)", got, limit)
	}
}
