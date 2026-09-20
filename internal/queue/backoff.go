package queue

import "time"

// Backoff returns how long to wait before retrying a job that has just
// failed its attempt-th attempt (1-based), using exponential backoff with
// full jitter (ADR-010).
//
// The ceiling doubles per attempt (base, 2*base, 4*base, ...) up to
// maxDelay; the returned delay is a random point between 0 and that
// ceiling. "Full jitter" means the whole range is randomized, not just a
// small fraction, so jobs that failed together do not retry together.
//
// rnd must return a value in [0, 1). It is a parameter so tests can pin
// the randomness and assert exact behavior.
func Backoff(attempt int, base, maxDelay time.Duration, rnd func() float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	ceiling := maxDelay
	// base << (attempt-1) is base * 2^(attempt-1). The shift is bounded so
	// a large attempt number cannot overflow int64 and wrap negative.
	if shift := attempt - 1; shift < 30 {
		if c := base << shift; c > 0 && c < maxDelay {
			ceiling = c
		}
	}

	return time.Duration(rnd() * float64(ceiling))
}
