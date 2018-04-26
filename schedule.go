package worker

import (
	"context"
	"time"

	"github.com/robfig/cron"
)

// ScheduleFunc is job wrapper for implement job run schedule
type ScheduleFunc func(context.Context, Job) Job

// ByTimer returns job wrapper func for run job each period duration
// after previous run completed
func ByTimer(period time.Duration) ScheduleFunc {
	return func(ctx context.Context, j Job) Job {
		return func(ctx context.Context) {
			timer := time.NewTimer(period)
			defer timer.Stop()

			for {
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

// ByTicker returns func wich run Worker by ticker each period duration
func ByTicker(period time.Duration) ScheduleFunc {
	return func(ctx context.Context, j Job) Job {
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
}

// ByCronSchedule returns job wrapper func for run job by cron schedule
// using robfig/cron parser for parse cron spec.
// If schedule spec not valid throw panic, shit happens.
func ByCronSchedule(schedule string) ScheduleFunc {
	s, err := cron.Parse(schedule)
	if err != nil {
		panic("parse cron spec fatal error: " + err.Error())
	}

	return func(ctx context.Context, job Job) Job {
		return func(ctx context.Context) {
			now := time.Now()
			timer := time.NewTimer(s.Next(now).Sub(now))
			defer timer.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					job(ctx)
					now = time.Now()
					timer.Reset(s.Next(now).Sub(now))
				}
			}
		}
	}
}
