package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
)

func main() {
	var (
		r1 int32 = 1
		r2 int32 = 2
		r3 int32 = 3
		r4 int32 = 4
	)

	job1 := incrementJobFunc("job1", &r1, -1)
	job2 := incrementJobFunc("job2", &r2, -1)
	job3 := incrementJobFunc("job3", &r3, -1)
	job4 := incrementJobFunc("job4", &r4, -1)

	// custom schedule, until 0
	scheduleFunc := func(target *int32) func(ctx context.Context, j worker.Job) worker.Job {
		return func(ctx context.Context, j worker.Job) worker.Job {
			return func(ctx context.Context) {
				for atomic.LoadInt32(target) > 0 {
					j(ctx)
				}
			}
		}
	}

	// custom lock func, timeout before run
	lockFunc := func(timeout time.Duration) func(ctx context.Context, j worker.Job) worker.Job {
		return func(ctx context.Context, j worker.Job) worker.Job {
			return func(ctx context.Context) {
				time.Sleep(timeout)
				j(ctx)
			}
		}
	}

	wrk1 := worker.New(job1).BySchedule(scheduleFunc(&r1))
	wrk2 := worker.New(job2).BySchedule(scheduleFunc(&r2)).WithLock(lockFunc(time.Second))
	wrk3 := worker.New(job3).BySchedule(scheduleFunc(&r3)).WithLock(lockFunc(time.Second))
	wrk4 := worker.New(job4).BySchedule(scheduleFunc(&r4))

	wk := worker.NewGroup()
	wk.Add(wrk1, wrk2, wrk3, wrk4)
	wk.Run()

	<-grace.ShutdownContext(context.Background()).Done()

	wk.Stop()
}

func incrementJobFunc(name string, target *int32, delta int32) func(context.Context) {
	return func(ctx context.Context) {
		log.Printf("%s start, int before: %d after %d", name, atomic.LoadInt32(target), atomic.AddInt32(target, delta))
	}
}
