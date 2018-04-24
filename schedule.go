package worker

import (
	"context"
	"time"

	"github.com/robfig/cron"
)

type ScheduleFunc func(context.Context, Job) Job

func ByTimer(period time.Duration) ScheduleFunc {
	return func(ctx context.Context, j Job) Job {
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
				default:
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
}
