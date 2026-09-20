// Package worker runs jobs from the durable queue. A Worker owns a
// bounded pool of slots; each slot repeatedly claims one job, runs the
// Handler on it while a heartbeat keeps its lease alive, and records the
// outcome. Alongside the slots, a reaper loop reclaims jobs whose workers
// died, and a listener turns LISTEN/NOTIFY into early wake-ups.
//
// Shutdown is a drain, not a kill: when Run's context is cancelled, slots
// stop claiming new jobs but let in-flight ones finish (up to
// Config.DrainTimeout), so a deploy or Ctrl-C does not throw away work.
package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ISHANT57/shipyard/internal/queue"
)

// Handler does the actual work for one claimed job. It must be idempotent
// (ADR-008: delivery is at-least-once). The context is cancelled if the
// worker loses the job's lease or the drain timeout expires; a Handler
// should stop promptly when that happens.
type Handler func(ctx context.Context, job queue.Job) error

// Config controls a Worker. Zero values fall back to defaults.
type Config struct {
	// ID prefixes every slot's worker ID ("<ID>/<slot>"), so the queue's
	// fencing check can tell slots apart.
	ID string
	// Concurrency is the number of slots, i.e. the maximum jobs in flight.
	Concurrency int
	// PollInterval is how long an idle slot waits before re-checking the
	// queue when no notification arrives. It is jittered by +/-50%.
	PollInterval time.Duration
	// ReapInterval is how often expired leases are reclaimed.
	ReapInterval time.Duration
	// ReapBatch caps how many expired jobs one reap pass handles.
	ReapBatch int
	// DrainTimeout bounds how long shutdown waits for in-flight jobs.
	DrainTimeout time.Duration
	// FinishTimeout bounds each Complete/Fail write after a job ends. It
	// uses its own deadline, not the run context, so results are still
	// recorded while draining.
	FinishTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.ID == "" {
		c.ID = "worker"
	}
	if c.Concurrency < 1 {
		c.Concurrency = 4
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.ReapInterval <= 0 {
		c.ReapInterval = 10 * time.Second
	}
	if c.ReapBatch < 1 {
		c.ReapBatch = 50
	}
	if c.DrainTimeout <= 0 {
		c.DrainTimeout = 30 * time.Second
	}
	if c.FinishTimeout <= 0 {
		c.FinishTimeout = 10 * time.Second
	}
	return c
}

// Worker runs queued jobs through a Handler.
type Worker struct {
	q      *queue.Queue
	handle Handler
	cfg    Config
	log    *slog.Logger
}

// New returns a Worker over q. Call Run to start it.
func New(q *queue.Queue, handle Handler, cfg Config, log *slog.Logger) *Worker {
	return &Worker{q: q, handle: handle, cfg: cfg.withDefaults(), log: log}
}

// Run starts the pool, the reaper and the listener, and blocks until ctx
// is cancelled and every in-flight job has finished (or been abandoned at
// the drain timeout). It returns nil after a clean drain.
func (w *Worker) Run(ctx context.Context) error {
	// execCtx is what handlers see. It deliberately does NOT inherit ctx's
	// cancellation: cancelling ctx begins the drain, and in-flight jobs
	// must keep running through it. execCancel fires only when the drain
	// timeout expires.
	execCtx, execCancel := context.WithCancel(context.WithoutCancel(ctx))
	defer execCancel()

	// One buffered token per slot: a notification wakes at most as many
	// idle slots as could actually take a job.
	wake := make(chan struct{}, w.cfg.Concurrency)

	var helpers sync.WaitGroup
	helpers.Add(2)
	go func() { defer helpers.Done(); w.reapLoop(ctx) }()
	go func() { defer helpers.Done(); w.listenLoop(ctx, wake) }()

	var slots sync.WaitGroup
	for i := 0; i < w.cfg.Concurrency; i++ {
		slots.Add(1)
		go func(slot int) {
			defer slots.Done()
			w.slotLoop(ctx, execCtx, fmt.Sprintf("%s/%d", w.cfg.ID, slot), wake)
		}(i)
	}

	w.log.Info("worker started", "id", w.cfg.ID, "concurrency", w.cfg.Concurrency)

	<-ctx.Done()
	w.log.Info("shutdown requested: draining in-flight jobs", "timeout", w.cfg.DrainTimeout.String())

	drainTimer := time.AfterFunc(w.cfg.DrainTimeout, func() {
		w.log.Warn("drain timeout reached: cancelling in-flight jobs", "timeout", w.cfg.DrainTimeout.String())
		execCancel()
	})
	defer drainTimer.Stop()

	slots.Wait()
	helpers.Wait()
	w.log.Info("worker stopped")
	return nil
}

// slotLoop is one pool slot: claim, run, record, repeat.
func (w *Worker) slotLoop(ctx, execCtx context.Context, workerID string, wake <-chan struct{}) {
	for ctx.Err() == nil {
		job, err := w.q.Claim(ctx, workerID)
		switch {
		case err == nil:
			// Note: if ctx is cancelled between the database committing a
			// claim and Claim returning, the worker sees an error for a job
			// it does hold. The lease simply expires and the reaper
			// reclaims it — at-least-once, not a lost job.
			w.runJob(execCtx, workerID, job)
			continue
		case errors.Is(err, queue.ErrNoJob):
			// Fall through to the idle wait below.
		case ctx.Err() != nil:
			return
		default:
			w.log.Error("claim failed", "worker", workerID, "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-wake:
		case <-time.After(jitter(w.cfg.PollInterval)):
		}
	}
}

// runJob executes one claimed job start to finish: heartbeat while the
// handler runs, then Complete or Fail — unless the lease was lost, in
// which case the result is discarded because another worker now owns it.
func (w *Worker) runJob(execCtx context.Context, workerID string, job queue.Job) {
	log := w.log.With("job", job.ID, "attempt", job.Attempt, "worker", workerID)
	log.Info("job started")
	started := time.Now()

	jobCtx, cancel := context.WithCancel(execCtx)
	defer cancel()

	var leaseLost atomic.Bool
	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		w.heartbeat(jobCtx, cancel, &leaseLost, job, workerID, log)
	}()

	handlerErr := w.safeHandle(jobCtx, job)

	// Stop the heartbeat and wait for it, so it cannot race the write below.
	cancel()
	<-heartbeatDone

	if leaseLost.Load() {
		log.Warn("lease lost: discarding result; another worker now owns this job")
		return
	}

	// Recording the outcome must survive a cancelled run context.
	finishCtx, finishCancel := context.WithTimeout(context.WithoutCancel(execCtx), w.cfg.FinishTimeout)
	defer finishCancel()

	if handlerErr == nil {
		if err := w.q.Complete(finishCtx, job, workerID); err != nil {
			w.logFinishError(log, "complete", err)
			return
		}
		log.Info("job succeeded", "duration", time.Since(started).String())
		return
	}

	if err := w.q.Fail(finishCtx, job, workerID, handlerErr.Error()); err != nil {
		w.logFinishError(log, "fail", err)
		return
	}
	log.Warn("job failed", "duration", time.Since(started).String(), "error", handlerErr)
}

func (w *Worker) logFinishError(log *slog.Logger, op string, err error) {
	if errors.Is(err, queue.ErrLeaseLost) {
		log.Warn("lease lost before result could be recorded", "op", op)
		return
	}
	// The lease will expire and the reaper will retry the job.
	log.Error("could not record job result", "op", op, "error", err)
}

// heartbeat extends the job's lease at a third of the lease length, so a
// couple of missed beats still leave margin. If the queue says the lease
// is gone, it flags that and cancels the job's context so the handler
// stops working on a job it no longer owns.
func (w *Worker) heartbeat(ctx context.Context, cancelJob context.CancelFunc, leaseLost *atomic.Bool, job queue.Job, workerID string, log *slog.Logger) {
	interval := w.q.Lease() / 3
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := w.q.Heartbeat(ctx, job, workerID)
			switch {
			case err == nil:
			case errors.Is(err, queue.ErrLeaseLost):
				leaseLost.Store(true)
				cancelJob()
				return
			case ctx.Err() != nil:
				return
			default:
				// A database blip: keep trying. The lease has margin.
				log.Warn("heartbeat failed", "error", err)
			}
		}
	}
}

// safeHandle runs the handler and converts a panic into an ordinary
// error, so one bad job fails (and eventually dead-letters) instead of
// crashing the whole worker and every job running beside it.
func (w *Worker) safeHandle(ctx context.Context, job queue.Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return w.handle(ctx, job)
}

// reapLoop periodically reclaims jobs whose leases expired. Every worker
// runs one; SKIP LOCKED inside Reap keeps them off each other's rows.
func (w *Worker) reapLoop(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.ReapInterval)
	defer ticker.Stop()

	for {
		if n, err := w.q.Reap(ctx, w.cfg.ReapBatch); err != nil {
			if ctx.Err() != nil {
				return
			}
			w.log.Error("reap failed", "error", err)
		} else if n > 0 {
			w.log.Warn("reclaimed jobs with expired leases", "count", n)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// listenLoop keeps a LISTEN subscription alive, reconnecting after
// failures. It is only an optimization: while it is down, slots still
// find jobs by polling.
func (w *Worker) listenLoop(ctx context.Context, wake chan<- struct{}) {
	for ctx.Err() == nil {
		err := w.q.Listen(ctx, wake)
		if ctx.Err() != nil {
			return
		}
		w.log.Warn("notification listener stopped; falling back to polling until it reconnects", "error", err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// jitter spreads d by +/-50% so idle slots and separate workers do not
// poll the database in lockstep.
func jitter(d time.Duration) time.Duration {
	return time.Duration(float64(d) * (0.5 + rand.Float64()))
}
