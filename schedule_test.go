package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestByCustomSchedule(t *testing.T) {
	Convey("Given target int == 5, decrement job and custom schedule (until int > 0)", t, func() {
		var (
			i   int32 = 5
			res       = make(chan struct{})
			job       = func(ctx context.Context) {
				atomic.AddInt32(&i, -1)
			}
		)

		schedule := func(ctx context.Context, j worker.Job) worker.Job {
			return func(ctx context.Context) {
				for atomic.LoadInt32(&i) > 0 {
					j(ctx)
				}
				res <- struct{}{}
			}
		}

		Convey("When run worker", func() {
			go worker.New(job).
				BySchedule(schedule).
				Run(context.Background())

			Convey("Job should be executed 5 times", func() {
				checkResulChannel(res)
				So(atomic.LoadInt32(&i), ShouldEqual, 0)
			})
		})
	})
}

func TestByTimer(t *testing.T) {

	Convey("Given job who send to result channel execution time and sleep for 1s", t, func() {
		res := make(chan time.Time)
		job := createWriterJob(time.Second, res)

		Convey("When create worker and run with 1s timer", func() {
			wrk := worker.
				New(job).
				ByTimer(time.Second)

			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)
			expectedNextExecutionTime := time.Now().Add(time.Second)

			Convey("job should be executed after 1s from previous run", func() {
				timer := time.NewTimer(2 * time.Second)
				defer timer.Stop()

				for i := 0; i < 3; i++ {
					select {
					case r := <-res:
						So(int64(expectedNextExecutionTime.Sub(r).Seconds()), ShouldEqual, 0)
						expectedNextExecutionTime = r.Add(time.Second)
						timer.Reset(2 * time.Second)
					case <-timer.C:
						So(false, ShouldBeTrue)
					}
				}
			})

			Convey("When cancel context", func() {
				cancel()

				Convey("job execution should be stopped", func() {
					timer := time.NewTimer(2 * time.Second)
					defer timer.Stop()

					select {
					case <-res:
						So(false, ShouldBeTrue)
					case <-timer.C:
						So(true, ShouldBeTrue)
					}
				})
			})
		})
	})
}

func TestByTicker(t *testing.T) {

	Convey("Given job who send to result channel execution time and sleep for 1s", t, func() {
		res := make(chan time.Time)
		job := createWriterJob(time.Second, res)

		Convey("When create worker and run with 1s ticker", func() {
			wrk := worker.
				New(job).
				ByTicker(time.Second)

			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)
			expectedNextExecutionTime := time.Now().Add(time.Second)

			Convey("job should be executed every 1s", func() {
				timer := time.NewTimer(2 * time.Second)
				defer timer.Stop()

				for i := 0; i < 3; i++ {
					select {
					case r := <-res:
						So(int64(expectedNextExecutionTime.Sub(r).Seconds()), ShouldEqual, 0)
						expectedNextExecutionTime = r.Add(time.Second)
						timer.Reset(2 * time.Second)
					case <-timer.C:
						So(false, ShouldBeTrue)
					}
				}
			})

			Convey("When cancel context", func() {
				cancel()

				Convey("job execution should be stopped", func() {
					timer := time.NewTimer(time.Second)
					defer timer.Stop()

					select {
					case <-res:
						So(false, ShouldBeTrue)
					case <-timer.C:
						So(true, ShouldBeTrue)
					}
				})
			})
		})
	})

	Convey("Given job who send to result channel execution time", t, func() {
		res := make(chan time.Time)
		job := func(ctx context.Context) {
			res <- time.Now()
		}

		Convey("When run with micro tiimeout ticker", func() {
			wrk := worker.New(job).ByTicker(time.Millisecond)

			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)

			Convey("Cancel context should stop job on next run", func() {
				time.Sleep(time.Second)
				cancel()
				<-res

				t := time.NewTimer(2 * time.Second)
				defer t.Stop()
				select {
				case <-res:
					So("gotten job result after stop", ShouldBeFalse)
				case <-t.C:
					So(true, ShouldBeTrue)
				}
			})
		})
	})
}

func TestByCronSchedule(t *testing.T) {

	Convey("Given job who send to result channel execution time and sleep for 1s", t, func() {
		res := make(chan time.Time)
		job := createWriterJob(time.Microsecond, res)

		Convey("When create worker with incorrect cron spec should panic", func() {
			So(func() { worker.New(job).ByCronSpec("завтра") }, ShouldPanic)
			So(func() { worker.New(job).ByCronSpec("@today") }, ShouldPanic)
			So(func() { worker.New(job).ByCronSpec("*") }, ShouldPanic)
		})

		Convey("When create worker and run with 1s cron schedule", func() {
			wrk := worker.
				New(job).
				ByCronSpec("@every 1s")

			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)
			expectedNextExecutionTime := time.Now().Add(time.Second)

			Convey("job should be executed every 1s", func() {
				timer := time.NewTimer(2 * time.Second)
				defer timer.Stop()

				for i := 0; i < 3; i++ {
					select {
					case r := <-res:
						So(int64(expectedNextExecutionTime.Sub(r).Seconds()), ShouldEqual, 0)
						expectedNextExecutionTime = r.Add(time.Second)
						timer.Reset(2 * time.Second)
					case <-timer.C:
						So(false, ShouldBeTrue)
					}
				}
			})

			Convey("When cancel context", func() {
				cancel()

				Convey("job execution should be stopped", func() {
					timer := time.NewTimer(time.Second)
					defer timer.Stop()

					select {
					case <-res:
						So(false, ShouldBeTrue)
					case <-timer.C:
						So(true, ShouldBeTrue)
					}
				})
			})
		})
	})
}
func createWriterJob(sleep time.Duration, ch chan time.Time) worker.Job {
	return func(ctx context.Context) {
		select {
		case ch <- time.Now():
		case <-ctx.Done():
		}
	}
}
