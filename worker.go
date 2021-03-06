package worker

import (
	"context"
	"time"
)

// Job is target background job
type Job func(context.Context)

// ObserveFunc given execution job time duration seconds
type ObserveFunc func(float64)

// Worker is builder for job with optional schedule and exclusive control
type Worker struct {
	job         Job
	schedule    ScheduleFunc
	locker      LockFunc
	observe     ObserveFunc
	immediately bool
}

// New returns new worker with target job
func New(job Job) *Worker {
	return &Worker{
		job: job,
	}
}

// BySchedule set schedule wrapper func for job
func (w *Worker) BySchedule(s ScheduleFunc) *Worker {
	w.schedule = s
	return w
}

// ByTimer set schedule timer job wrapper with period
func (w *Worker) ByTimer(period time.Duration) *Worker {
	w.schedule = ByTimer(period)
	return w
}

// ByTicker set schedule ticker job wrapper with period
func (w *Worker) ByTicker(period time.Duration) *Worker {
	w.schedule = ByTicker(period)
	return w
}

// ByCronSpec set schedule job wrapper by cron spec
func (w *Worker) ByCronSpec(spec string) *Worker {
	w.schedule = ByCronSchedule(spec)
	return w
}

// WithLock set job lock wrapper
func (w *Worker) WithLock(l Locker) *Worker {
	w.locker = WithLock(l)
	return w
}

// SetImmediately set execute job on Run setting
func (w *Worker) SetImmediately(executeOnRun bool) *Worker {
	w.immediately = executeOnRun
	return w
}

// SetObserver set job duration observer
func (w *Worker) SetObserver(observe ObserveFunc) *Worker {
	w.observe = observe
	return w
}

// Run job, wrap job to metrics, lock and schedule wrappers
func (w *Worker) Run(ctx context.Context) {
	job := w.job

	if w.observe != nil {
		job = func(ctx context.Context) {
			start := time.Now()
			w.job(ctx)
			w.observe(time.Since(start).Seconds())
		}
	}

	if w.locker != nil {
		job = w.locker(ctx, job)
	}

	if w.immediately {
		job(ctx)

		if w.schedule == nil {
			return
		}

		// check context before run immediately job again
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	if w.schedule != nil {
		job = w.schedule(ctx, job)
	}

	job(ctx)
}
