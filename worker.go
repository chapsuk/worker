package worker

import (
	"context"
	"time"
)

type Job func(context.Context)

type MetricObserveFunc func(float64)

type Worker struct {
	job             Job
	schedule        ScheduleFunc
	locker          LockFunc
	metricsObserver MetricObserveFunc
}

func New(job Job) *Worker {
	return &Worker{
		job: job,
	}
}

func (w *Worker) BySchedule(s ScheduleFunc) *Worker {
	w.schedule = s
	return w
}

func (w *Worker) ByTimer(period time.Duration) *Worker {
	w.schedule = ByTimer(period)
	return w
}

func (w *Worker) ByTicker(period time.Duration) *Worker {
	w.schedule = ByTicker(period)
	return w
}

func (w *Worker) ByCronSpec(spec string) *Worker {
	w.schedule = ByCronSchedule(spec)
	return w
}

func (w *Worker) WithLock(l Locker) *Worker {
	w.locker = WithLock(l)
	return w
}

func (w *Worker) WithRedisLock(opts RedisLockOptions) *Worker {
	w.locker = WithRedisLock(opts)
	return w
}

func (w *Worker) WithBsmRedisLock(opts BsmRedisLockOptions) *Worker {
	w.locker = WithBsmRedisLock(opts)
	return w
}

func (w *Worker) WithMetrics(observe MetricObserveFunc) *Worker {
	w.metricsObserver = observe
	return w
}

func (w *Worker) Run(ctx context.Context) {
	job := w.job

	if w.metricsObserver != nil {
		job = func(ctx context.Context) {
			start := time.Now()
			w.job(ctx)
			w.metricsObserver(time.Since(start).Seconds())
		}
	}

	if w.locker != nil {
		job = w.locker(ctx, job)
	}

	if w.schedule != nil {
		job = w.schedule(ctx, job)
	}

	job(ctx)
}
