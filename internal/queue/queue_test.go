package queue

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ISHANT57/shipyard/internal/testdb"
)

// fixedRand pins jitter to the top of the range, so a failed job's
// backoff is exactly the ceiling and tests can reason about it.
func fixedRand() float64 { return 0.999999 }

func newTestQueue(t *testing.T, opts Options) (*Queue, *pgxpool.Pool) {
	t.Helper()
	dsn := testdb.NewPostgres(t)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if opts.Rand == nil {
		opts.Rand = fixedRand
	}
	return New(pool, opts), pool
}

// seedStage delegates to the shared helper in internal/testdb.
func seedStage(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	return testdb.SeedStage(t, pool)
}

func mustEnqueue(t *testing.T, q *Queue, stageID uuid.UUID) uuid.UUID {
	t.Helper()
	id, err := q.Enqueue(context.Background(), stageID)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	return id
}

func mustGet(t *testing.T, q *Queue, id uuid.UUID) Record {
	t.Helper()
	r, err := q.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return r
}

// makeClaimable pulls a job's run_after into the past, standing in for
// "the backoff delay has elapsed" without sleeping through it.
func makeClaimable(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`UPDATE jobs SET run_after = now() - interval '1 second' WHERE id = $1`, id); err != nil {
		t.Fatalf("makeClaimable: %v", err)
	}
}

func attemptStatus(t *testing.T, pool *pgxpool.Pool, jobID uuid.UUID, n int) (status string, errMsg *string) {
	t.Helper()
	if err := pool.QueryRow(context.Background(),
		`SELECT status, error FROM job_attempts WHERE job_id = $1 AND attempt_number = $2`, jobID, n,
	).Scan(&status, &errMsg); err != nil {
		t.Fatalf("attemptStatus: %v", err)
	}
	return status, errMsg
}

func TestBackoff(t *testing.T) {
	base, maxDelay := 5*time.Second, 5*time.Minute
	top := func() float64 { return 0.999999999 }
	zero := func() float64 { return 0 }

	cases := []struct {
		name    string
		attempt int
		rnd     func() float64
		want    func(d time.Duration) bool
	}{
		{"attempt 1 ceiling is base", 1, top, func(d time.Duration) bool { return d > 4*time.Second && d <= base }},
		{"attempt 2 ceiling doubles", 2, top, func(d time.Duration) bool { return d > 9*time.Second && d <= 2*base }},
		{"attempt 3 ceiling doubles again", 3, top, func(d time.Duration) bool { return d > 19*time.Second && d <= 4*base }},
		{"large attempt is capped", 40, top, func(d time.Duration) bool { return d > 4*time.Minute && d <= maxDelay }},
		{"attempt below 1 treated as 1", 0, top, func(d time.Duration) bool { return d <= base }},
		{"jitter can reach zero", 3, zero, func(d time.Duration) bool { return d == 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Backoff(tc.attempt, base, maxDelay, tc.rnd)
			if !tc.want(got) {
				t.Errorf("Backoff(%d) = %s, outside expected range", tc.attempt, got)
			}
		})
	}
}

func TestClaim_NoJobAvailable(t *testing.T) {
	q, _ := newTestQueue(t, Options{})

	_, err := q.Claim(context.Background(), "worker-a")
	if !errors.Is(err, ErrNoJob) {
		t.Errorf("Claim() error = %v, want ErrNoJob", err)
	}
}

func TestEnqueueClaimComplete(t *testing.T) {
	q, pool := newTestQueue(t, Options{})
	ctx := context.Background()
	id := mustEnqueue(t, q, seedStage(t, pool))

	job, err := q.Claim(ctx, "worker-a")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if job.ID != id || job.Attempt != 1 {
		t.Errorf("Claim() = %+v, want job %s attempt 1", job, id)
	}
	if r := mustGet(t, q, id); r.Status != "running" || r.LockedBy == nil || *r.LockedBy != "worker-a" {
		t.Errorf("after Claim, record = %+v, want running/worker-a", r)
	}
	if s, _ := attemptStatus(t, pool, id, 1); s != "running" {
		t.Errorf("attempt 1 status = %q, want running", s)
	}

	if err := q.Complete(ctx, job, "worker-a"); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if r := mustGet(t, q, id); r.Status != "succeeded" || r.LockedBy != nil {
		t.Errorf("after Complete, record = %+v, want succeeded and unlocked", r)
	}
	if s, _ := attemptStatus(t, pool, id, 1); s != "succeeded" {
		t.Errorf("attempt 1 status = %q, want succeeded", s)
	}

	// A finished job is not claimable again.
	if _, err := q.Claim(ctx, "worker-b"); !errors.Is(err, ErrNoJob) {
		t.Errorf("second Claim() error = %v, want ErrNoJob", err)
	}
}

func TestClaim_SkipsJobsNotYetDue(t *testing.T) {
	q, pool := newTestQueue(t, Options{})
	id := mustEnqueue(t, q, seedStage(t, pool))
	if _, err := pool.Exec(context.Background(),
		`UPDATE jobs SET run_after = now() + interval '1 hour' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}

	if _, err := q.Claim(context.Background(), "worker-a"); !errors.Is(err, ErrNoJob) {
		t.Errorf("Claim() error = %v, want ErrNoJob for a job due in the future", err)
	}
}

// TestClaim_SkipsRowsLockedByAnotherTransaction pins down what SKIP
// LOCKED actually buys: not correctness (plain row locks already prevent
// double-claims) but *not waiting*. Job A is locked by another
// transaction; Claim must go straight past it to job B instead of
// blocking on A or reporting "no job".
func TestClaim_SkipsRowsLockedByAnotherTransaction(t *testing.T) {
	q, pool := newTestQueue(t, Options{})
	stageID := seedStage(t, pool)
	idA := mustEnqueue(t, q, stageID)
	idB := mustEnqueue(t, q, stageID)

	bg := context.Background()
	holder, err := pool.Begin(bg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(bg) }()
	// A is first in claim order (older run_after); lock it and hold.
	if _, err := holder.Exec(bg, `SELECT 1 FROM jobs WHERE id = $1 FOR UPDATE`, idA); err != nil {
		t.Fatal(err)
	}

	// A short deadline turns "blocked forever" into a clear test failure.
	ctx, cancel := context.WithTimeout(bg, 3*time.Second)
	defer cancel()
	start := time.Now()
	job, err := q.Claim(ctx, "worker-a")
	if err != nil {
		t.Fatalf("Claim() error = %v (did it block on the locked row?)", err)
	}
	if job.ID != idB {
		t.Errorf("Claim() returned %s, want %s (the unlocked job)", job.ID, idB)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Claim() took %s, want near-instant", elapsed)
	}
}

func TestComplete_IsFenced(t *testing.T) {
	q, pool := newTestQueue(t, Options{})
	ctx := context.Background()
	id := mustEnqueue(t, q, seedStage(t, pool))
	job, err := q.Claim(ctx, "worker-a")
	if err != nil {
		t.Fatal(err)
	}

	// Wrong worker: rejected, job untouched.
	if err := q.Complete(ctx, job, "worker-b"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("Complete() by wrong worker error = %v, want ErrLeaseLost", err)
	}
	// Right worker, wrong attempt token: rejected.
	stale := job
	stale.Attempt = job.Attempt + 1
	if err := q.Complete(ctx, stale, "worker-a"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("Complete() with wrong attempt error = %v, want ErrLeaseLost", err)
	}
	if r := mustGet(t, q, id); r.Status != "running" {
		t.Errorf("status after rejected completes = %q, want running", r.Status)
	}

	// The real owner still succeeds, and a second Complete is rejected.
	if err := q.Complete(ctx, job, "worker-a"); err != nil {
		t.Fatalf("Complete() by owner error = %v", err)
	}
	if err := q.Complete(ctx, job, "worker-a"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("second Complete() error = %v, want ErrLeaseLost", err)
	}
}

func TestFail_RetriesWithBackoffThenReclaims(t *testing.T) {
	q, pool := newTestQueue(t, Options{BackoffBase: 5 * time.Second})
	ctx := context.Background()
	id := mustEnqueue(t, q, seedStage(t, pool))
	job, err := q.Claim(ctx, "worker-a")
	if err != nil {
		t.Fatal(err)
	}

	if err := q.Fail(ctx, job, "worker-a", "boom"); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}

	r := mustGet(t, q, id)
	if r.Status != "queued" || r.LockedBy != nil {
		t.Errorf("after Fail, record = %+v, want queued and unlocked", r)
	}
	if !r.RunAfter.After(time.Now().Add(2 * time.Second)) {
		t.Errorf("run_after = %s, want a backoff delay in the future", r.RunAfter)
	}
	if s, msg := attemptStatus(t, pool, id, 1); s != "failed" || msg == nil || *msg != "boom" {
		t.Errorf("attempt 1 = %q/%v, want failed/boom", s, msg)
	}

	// During the backoff window the job is not claimable...
	if _, err := q.Claim(ctx, "worker-b"); !errors.Is(err, ErrNoJob) {
		t.Errorf("Claim() during backoff error = %v, want ErrNoJob", err)
	}
	// ...and once it elapses, attempt 2 is claimed.
	makeClaimable(t, pool, id)
	retry, err := q.Claim(ctx, "worker-b")
	if err != nil {
		t.Fatalf("Claim() after backoff error = %v", err)
	}
	if retry.ID != id || retry.Attempt != 2 {
		t.Errorf("retry = %+v, want job %s attempt 2", retry, id)
	}
}

func TestFail_PoisonJobGoesToDeadLetter(t *testing.T) {
	q, pool := newTestQueue(t, Options{})
	ctx := context.Background()
	id := mustEnqueue(t, q, seedStage(t, pool))
	if _, err := pool.Exec(ctx, `UPDATE jobs SET max_attempts = 3 WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 3; attempt++ {
		makeClaimable(t, pool, id)
		job, err := q.Claim(ctx, "worker-a")
		if err != nil {
			t.Fatalf("attempt %d: Claim() error = %v", attempt, err)
		}
		if job.Attempt != attempt {
			t.Fatalf("attempt %d: got job.Attempt = %d", attempt, job.Attempt)
		}
		if err := q.Fail(ctx, job, "worker-a", "always fails"); err != nil {
			t.Fatalf("attempt %d: Fail() error = %v", attempt, err)
		}
	}

	if r := mustGet(t, q, id); r.Status != "dead" {
		t.Errorf("status after max_attempts failures = %q, want dead", r.Status)
	}
	makeClaimable(t, pool, id)
	if _, err := q.Claim(ctx, "worker-a"); !errors.Is(err, ErrNoJob) {
		t.Errorf("Claim() on a dead job error = %v, want ErrNoJob", err)
	}
}

func TestHeartbeat_ExtendsLeaseAndIsFenced(t *testing.T) {
	q, pool := newTestQueue(t, Options{Lease: 2 * time.Second})
	ctx := context.Background()
	mustEnqueue(t, q, seedStage(t, pool))
	job, err := q.Claim(ctx, "worker-a")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)
	if err := q.Heartbeat(ctx, job, "worker-a"); err != nil {
		t.Fatalf("Heartbeat() error = %v", err)
	}
	var until time.Time
	if err := pool.QueryRow(ctx, `SELECT locked_until FROM jobs WHERE id = $1`, job.ID).Scan(&until); err != nil {
		t.Fatal(err)
	}
	if !until.After(job.LockedUntil) {
		t.Errorf("locked_until after heartbeat = %s, want later than original %s", until, job.LockedUntil)
	}

	if err := q.Heartbeat(ctx, job, "worker-b"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("Heartbeat() by non-owner error = %v, want ErrLeaseLost", err)
	}
}

// TestReap_CrashedWorkerJobIsReclaimed simulates a worker that claims a
// job and dies: nothing ever calls Complete or Fail. After the lease
// expires the reaper reclaims the job, another worker finishes it, and
// the original worker — should it wake up later — is fenced out.
func TestReap_CrashedWorkerJobIsReclaimed(t *testing.T) {
	q, pool := newTestQueue(t, Options{Lease: 100 * time.Millisecond})
	ctx := context.Background()
	id := mustEnqueue(t, q, seedStage(t, pool))

	crashed, err := q.Claim(ctx, "worker-crashed")
	if err != nil {
		t.Fatal(err)
	}

	// Lease still valid: nothing to reap.
	if n, err := q.Reap(ctx, 10); err != nil || n != 0 {
		t.Fatalf("Reap() before expiry = %d, %v; want 0, nil", n, err)
	}

	time.Sleep(250 * time.Millisecond)
	n, err := q.Reap(ctx, 10)
	if err != nil || n != 1 {
		t.Fatalf("Reap() after expiry = %d, %v; want 1, nil", n, err)
	}
	if s, msg := attemptStatus(t, pool, id, 1); s != "failed" || msg == nil || *msg != "lease expired" {
		t.Errorf("attempt 1 = %q/%v, want failed/lease expired", s, msg)
	}

	makeClaimable(t, pool, id)
	rescuer, err := q.Claim(ctx, "worker-rescuer")
	if err != nil {
		t.Fatalf("Claim() after reap error = %v", err)
	}
	if rescuer.Attempt != 2 {
		t.Errorf("rescuer attempt = %d, want 2", rescuer.Attempt)
	}

	// The crashed worker wakes up: every action it tries is rejected.
	if err := q.Complete(ctx, crashed, "worker-crashed"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("stale Complete() error = %v, want ErrLeaseLost", err)
	}
	if err := q.Heartbeat(ctx, crashed, "worker-crashed"); !errors.Is(err, ErrLeaseLost) {
		t.Errorf("stale Heartbeat() error = %v, want ErrLeaseLost", err)
	}

	if err := q.Complete(ctx, rescuer, "worker-rescuer"); err != nil {
		t.Fatalf("rescuer Complete() error = %v", err)
	}
	if r := mustGet(t, q, id); r.Status != "succeeded" {
		t.Errorf("final status = %q, want succeeded", r.Status)
	}
}

// TestConcurrentWorkers_EachJobClaimedExactlyOnce is the Phase 04
// acceptance test: many workers race over many jobs, and every job must
// be claimed once and completed once — no double-claims, no losses.
func TestConcurrentWorkers_EachJobClaimedExactlyOnce(t *testing.T) {
	const (
		workers = 8
		jobs    = 60
	)
	q, pool := newTestQueue(t, Options{})
	ctx := context.Background()

	stageID := seedStage(t, pool)
	want := make(map[uuid.UUID]bool, jobs)
	for i := 0; i < jobs; i++ {
		want[mustEnqueue(t, q, stageID)] = true
	}

	var (
		mu      sync.Mutex
		claimed = make(map[uuid.UUID]int)
		wg      sync.WaitGroup
	)
	errs := make(chan error, workers)

	for w := 0; w < workers; w++ {
		workerID := uuid.NewString()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				job, err := q.Claim(ctx, workerID)
				if errors.Is(err, ErrNoJob) {
					return
				}
				if err != nil {
					errs <- err
					return
				}
				mu.Lock()
				claimed[job.ID]++
				mu.Unlock()
				if err := q.Complete(ctx, job, workerID); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("worker error: %v", err)
	}

	if len(claimed) != jobs {
		t.Errorf("distinct jobs claimed = %d, want %d", len(claimed), jobs)
	}
	for id, n := range claimed {
		if !want[id] {
			t.Errorf("claimed unknown job %s", id)
		}
		if n != 1 {
			t.Errorf("job %s claimed %d times, want exactly 1", id, n)
		}
	}

	var succeeded, attempts int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE status = 'succeeded'`).Scan(&succeeded); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM job_attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if succeeded != jobs || attempts != jobs {
		t.Errorf("succeeded = %d, attempts = %d; want %d each", succeeded, attempts, jobs)
	}
}
