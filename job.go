package job

import (
	"context"
	"time"

	"github.com/chapsuk/wait"
)

// Job is any func
type Job func(context.Context)

// Before returns func with ordered calls before and orig funcs
func Before(orig Job, before Job) Job {
	return func(ctx context.Context) {
		before(ctx)
		orig(ctx)
	}
}

// After returns func with ordered calls orig and after funcs
func After(orig Job, after Job) Job {
	return func(ctx context.Context) {
		orig(ctx)
		after(ctx)
	}
}

// ByTicker returns func wich run job by ticker each period duration
func ByTicker(period time.Duration, j Job) Job {
	return func(ctx context.Context) {
		ticker := time.NewTicker(period)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					j(ctx)
				}
			}
		}
	}
}

// ByTicker returns func wich run job by timer each period duration after previous run
func ByTimer(period time.Duration, j Job) Job {
	return func(ctx context.Context) {
		timer := time.NewTimer(period)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					j(ctx)
					timer.Reset(period)
				}
			}
		}
	}
}

// Locker interface
type Locker interface {
	Lock() error
	Unlock()
}

// WithLock returns func with call job in lock
func WithLock(l Locker, j Job) Job {
	return func(ctx context.Context) {
		if err := l.Lock(); err != nil {
			return
		}
		defer l.Unlock()

		j(ctx)
	}
}

// Many returns func wich run job count times and wait until all goroutines stopped
func Many(count int, j Job) Job {
	return func(ctx context.Context) {
		wg := wait.Group{}
		for i := 0; i < count; i++ {
			wg.AddWithContext(ctx, j)
		}
		wg.Wait()
	}
}
