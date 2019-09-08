package worker

import (
	"context"
)

// LockFunc is job wrapper for control exclusive execution
type LockFunc func(context.Context, Job) Job

// Locker interface
type Locker interface {
	// Lock acquire lock for job, returns error when the job should not be started
	Lock() error
	// Unlock release acquired lock
	Unlock()
}

// WithLock returns func with call Worker in lock
func WithLock(l Locker) LockFunc {
	return func(ctx context.Context, j Job) Job {
		return func(ctx context.Context) {
			if err := l.Lock(); err != nil {
				return
			}
			defer l.Unlock()
			j(ctx)
		}
	}
}
