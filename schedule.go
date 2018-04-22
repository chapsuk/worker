package worker

import (
	"context"
	"time"

	"github.com/robfig/cron"
)

// ByTicker returns func wich run Worker by ticker each period duration
func ByTicker(w Worker, period time.Duration) Worker {
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
func ByTimer(w Worker, period time.Duration) Worker {
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

func ByCronSchedule(w Worker, schedule string) (Worker, error) {
	s, err := cron.Parse(schedule)
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context) {
		now := time.Now()

		timer := time.NewTimer(s.Next(now).Sub(now))
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			w(ctx)
		}
	}, nil
}
