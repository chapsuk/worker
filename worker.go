package worker

import (
	"context"
	"time"

	"github.com/chapsuk/wait"
)

// Worker is any func
type Worker func(context.Context)

// ByTicker returns func wich run Worker by ticker each period duration
func ByTicker(period time.Duration, w Worker) Worker {
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
					w(ctx)
				}
			}
		}
	}
}

// ByTicker returns func wich run Worker by timer each period duration after previous run
func ByTimer(period time.Duration, w Worker) Worker {
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
					w(ctx)
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

// WithLock returns func with call Worker in lock
func WithLock(l Locker, w Worker) Worker {
	return func(ctx context.Context) {
		if err := l.Lock(); err != nil {
			return
		}
		defer l.Unlock()

		w(ctx)
	}
}

// Many returns func wich run Worker count times and wait until all goroutines stopped
func Many(count int, w Worker) Worker {
	return func(ctx context.Context) {
		wg := wait.Group{}
		for i := 0; i < count; i++ {
			wg.AddWithContext(ctx, w)
		}
		wg.Wait()
	}
}
