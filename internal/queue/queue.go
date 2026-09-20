// Package queue is Shipyard's durable job queue, built on the jobs table
// in PostgreSQL. Delivery is at-least-once with fenced completion
// (ADR-008); jobs are claimed with SELECT ... FOR UPDATE SKIP LOCKED
// (ADR-009); failures retry with full-jitter backoff and end in a
// dead-letter status (ADR-010).
//
// The lifecycle of one job:
//
//	Enqueue -> queued --Claim--> running --Complete--> succeeded
//	                                 |--Fail/Reap--> queued (retry, run_after in the future)
//	                                 '--Fail/Reap at max_attempts--> dead
package queue

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NotifyChannel is the LISTEN/NOTIFY channel Enqueue signals on. It is a
// latency hint only (ADR-009): correctness never depends on it.
const NotifyChannel = "shipyard_jobs"

var (
	// ErrNoJob is returned by Claim when no job is currently claimable.
	ErrNoJob = errors.New("queue: no job available")

	// ErrLeaseLost is returned when a worker acts on a job it no longer
	// owns: its lease expired and the job was reclaimed, or the job was
	// already finished. The worker must stop and discard its result.
	ErrLeaseLost = errors.New("queue: lease lost")
)

// Options tunes queue behavior. Zero values fall back to defaults.
type Options struct {
	// Lease is how long a claim stays valid without a heartbeat.
	Lease time.Duration
	// BackoffBase and BackoffMax shape retry delays (see Backoff).
	BackoffBase time.Duration
	BackoffMax  time.Duration
	// Rand returns a value in [0, 1) for backoff jitter. Tests inject a
	// fixed function; production uses math/rand/v2.
	Rand func() float64
}

func (o Options) withDefaults() Options {
	if o.Lease <= 0 {
		o.Lease = 30 * time.Second
	}
	if o.BackoffBase <= 0 {
		o.BackoffBase = 5 * time.Second
	}
	if o.BackoffMax <= 0 {
		o.BackoffMax = 5 * time.Minute
	}
	if o.Rand == nil {
		o.Rand = rand.Float64
	}
	return o
}

// Queue is a handle on the job queue. It is safe for concurrent use.
type Queue struct {
	pool *pgxpool.Pool
	opts Options
}

// New returns a Queue over pool.
func New(pool *pgxpool.Pool, opts Options) *Queue {
	return &Queue{pool: pool, opts: opts.withDefaults()}
}

// Lease returns the configured lease duration, so workers can pick a
// heartbeat interval from it.
func (q *Queue) Lease() time.Duration { return q.opts.Lease }

// Job is a claimed unit of work. Attempt is the fencing token: every
// later call about this claim must present the same WorkerID and Attempt.
type Job struct {
	ID          uuid.UUID
	StageID     uuid.UUID
	Attempt     int
	MaxAttempts int
	LockedUntil time.Time
}

// Record is a read-only view of a job row, for inspection and tests.
type Record struct {
	ID          uuid.UUID
	Status      string
	Attempt     int
	MaxAttempts int
	RunAfter    time.Time
	LockedBy    *string
}

// Enqueue adds a queued job for stage and signals waiting workers.
func (q *Queue) Enqueue(ctx context.Context, stageID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := pgx.BeginFunc(ctx, q.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`INSERT INTO jobs (stage_id) VALUES ($1) RETURNING id`, stageID,
		).Scan(&id); err != nil {
			return err
		}
		// NOTIFY is delivered only when this transaction commits, so a
		// woken worker always finds the row.
		_, err := tx.Exec(ctx, `SELECT pg_notify($1, $2)`, NotifyChannel, id.String())
		return err
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("queue: enqueue: %w", err)
	}
	return id, nil
}

// claimSQL is the heart of the queue (ADR-009). The inner SELECT locks
// one claimable row; SKIP LOCKED makes concurrent workers pass over rows
// another worker has locked instead of waiting, so no two workers can
// claim the same job and none blocks another.
const claimSQL = `
UPDATE jobs
SET status = 'running',
    locked_by = $1,
    attempt = attempt + 1,
    locked_until = now() + make_interval(secs => $2),
    updated_at = now()
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'queued' AND run_after <= now()
    ORDER BY run_after
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id, stage_id, attempt, max_attempts, locked_until`

// Claim leases the next claimable job to workerID. It returns ErrNoJob
// when nothing is claimable.
func (q *Queue) Claim(ctx context.Context, workerID string) (Job, error) {
	var job Job
	err := pgx.BeginFunc(ctx, q.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, claimSQL, workerID, q.opts.Lease.Seconds()).
			Scan(&job.ID, &job.StageID, &job.Attempt, &job.MaxAttempts, &job.LockedUntil)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNoJob
		}
		if err != nil {
			return err
		}
		// Every attempt is recorded, so a duplicate run is visible (ADR-008).
		_, err = tx.Exec(ctx,
			`INSERT INTO job_attempts (job_id, attempt_number, status) VALUES ($1, $2, 'running')`,
			job.ID, job.Attempt)
		return err
	})
	if errors.Is(err, ErrNoJob) {
		return Job{}, ErrNoJob
	}
	if err != nil {
		return Job{}, fmt.Errorf("queue: claim: %w", err)
	}
	return job, nil
}

// Heartbeat extends the lease on a job the caller still owns.
func (q *Queue) Heartbeat(ctx context.Context, job Job, workerID string) error {
	tag, err := q.pool.Exec(ctx,
		`UPDATE jobs
		 SET locked_until = now() + make_interval(secs => $4), updated_at = now()
		 WHERE id = $1 AND locked_by = $2 AND attempt = $3 AND status = 'running'`,
		job.ID, workerID, job.Attempt, q.opts.Lease.Seconds())
	if err != nil {
		return fmt.Errorf("queue: heartbeat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLeaseLost
	}
	return nil
}

// Complete marks the job succeeded. It is fenced: it only takes effect
// if workerID and job.Attempt still own the job (ADR-008).
func (q *Queue) Complete(ctx context.Context, job Job, workerID string) error {
	err := pgx.BeginFunc(ctx, q.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE jobs
			 SET status = 'succeeded', locked_by = NULL, locked_until = NULL, updated_at = now()
			 WHERE id = $1 AND locked_by = $2 AND attempt = $3 AND status = 'running'`,
			job.ID, workerID, job.Attempt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrLeaseLost
		}
		_, err = tx.Exec(ctx,
			`UPDATE job_attempts SET status = 'succeeded', finished_at = now()
			 WHERE job_id = $1 AND attempt_number = $2`,
			job.ID, job.Attempt)
		return err
	})
	if errors.Is(err, ErrLeaseLost) {
		return ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("queue: complete: %w", err)
	}
	return nil
}

// Fail records a failed attempt. The job is retried after a jittered
// backoff, or moved to the dead-letter status when attempts are
// exhausted. Fenced like Complete.
func (q *Queue) Fail(ctx context.Context, job Job, workerID, reason string) error {
	err := pgx.BeginFunc(ctx, q.pool, func(tx pgx.Tx) error {
		var attempt, maxAttempts int
		err := tx.QueryRow(ctx,
			`SELECT attempt, max_attempts FROM jobs
			 WHERE id = $1 AND locked_by = $2 AND attempt = $3 AND status = 'running'
			 FOR UPDATE`,
			job.ID, workerID, job.Attempt).Scan(&attempt, &maxAttempts)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLeaseLost
		}
		if err != nil {
			return err
		}
		return q.failLocked(ctx, tx, job.ID, attempt, maxAttempts, reason)
	})
	if errors.Is(err, ErrLeaseLost) {
		return ErrLeaseLost
	}
	if err != nil {
		return fmt.Errorf("queue: fail: %w", err)
	}
	return nil
}

// failLocked applies the retry policy to a job row the caller already
// holds locked inside tx. Fail and Reap share it, so a crashed worker and
// a failed worker are handled by exactly the same rules (ADR-010).
func (q *Queue) failLocked(ctx context.Context, tx pgx.Tx, id uuid.UUID, attempt, maxAttempts int, reason string) error {
	if attempt >= maxAttempts {
		if _, err := tx.Exec(ctx,
			`UPDATE jobs SET status = 'dead', locked_by = NULL, locked_until = NULL, updated_at = now()
			 WHERE id = $1`, id); err != nil {
			return err
		}
	} else {
		delay := Backoff(attempt, q.opts.BackoffBase, q.opts.BackoffMax, q.opts.Rand)
		if _, err := tx.Exec(ctx,
			`UPDATE jobs
			 SET status = 'queued', run_after = now() + make_interval(secs => $2),
			     locked_by = NULL, locked_until = NULL, updated_at = now()
			 WHERE id = $1`, id, delay.Seconds()); err != nil {
			return err
		}
	}

	_, err := tx.Exec(ctx,
		`UPDATE job_attempts SET status = 'failed', error = $3, finished_at = now()
		 WHERE job_id = $1 AND attempt_number = $2`,
		id, attempt, reason)
	return err
}

// Reap reclaims running jobs whose lease has expired, treating each as a
// failed attempt. It handles at most limit jobs and returns how many it
// reclaimed. Safe to run from several workers at once: SKIP LOCKED keeps
// them off each other's rows.
func (q *Queue) Reap(ctx context.Context, limit int) (int, error) {
	reaped := 0
	err := pgx.BeginFunc(ctx, q.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, attempt, max_attempts FROM jobs
			 WHERE status = 'running' AND locked_until < now()
			 ORDER BY locked_until
			 FOR UPDATE SKIP LOCKED
			 LIMIT $1`, limit)
		if err != nil {
			return err
		}
		type expired struct {
			id                   uuid.UUID
			attempt, maxAttempts int
		}
		var batch []expired
		for rows.Next() {
			var e expired
			if err := rows.Scan(&e.id, &e.attempt, &e.maxAttempts); err != nil {
				return err
			}
			batch = append(batch, e)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// The rows are fully read before any further statement runs on tx:
		// a pgx connection cannot interleave a second query with an open
		// result set.
		for _, e := range batch {
			if err := q.failLocked(ctx, tx, e.id, e.attempt, e.maxAttempts, "lease expired"); err != nil {
				return err
			}
			reaped++
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("queue: reap: %w", err)
	}
	return reaped, nil
}

// Listen blocks, forwarding a signal on wake each time Enqueue announces
// a new job, until ctx is cancelled or the connection fails. Sends never
// block: if wake is full a wake-up is already pending, which is enough.
//
// This is the latency hint from ADR-009. A worker that misses a
// notification (reconnecting, busy) still finds the job on its next poll.
func (q *Queue) Listen(ctx context.Context, wake chan<- struct{}) error {
	pooled, err := q.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("queue: listen: acquire: %w", err)
	}
	// Hijack takes the connection out of the pool for good. A connection
	// that has run LISTEN must never be handed to another caller, who
	// would inherit a stray subscription.
	conn := pooled.Hijack()
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	if _, err := conn.Exec(ctx, "LISTEN "+NotifyChannel); err != nil {
		return fmt.Errorf("queue: listen: %w", err)
	}
	for {
		if _, err := conn.WaitForNotification(ctx); err != nil {
			return fmt.Errorf("queue: listen: %w", err)
		}
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

// Get returns the current state of a job, or pgx.ErrNoRows-wrapped error
// if it does not exist.
func (q *Queue) Get(ctx context.Context, id uuid.UUID) (Record, error) {
	var r Record
	err := q.pool.QueryRow(ctx,
		`SELECT id, status, attempt, max_attempts, run_after, locked_by FROM jobs WHERE id = $1`, id,
	).Scan(&r.ID, &r.Status, &r.Attempt, &r.MaxAttempts, &r.RunAfter, &r.LockedBy)
	if err != nil {
		return Record{}, fmt.Errorf("queue: get: %w", err)
	}
	return r, nil
}
