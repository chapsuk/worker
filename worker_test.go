package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMetricsObserver(t *testing.T) {
	Convey("Giiven Observ func and job with 50ms timeout", t, func() {
		obs := func(d float64) {
			So(d, ShouldBeGreaterThan, 0.05)
		}
		job := func(ctx context.Context) {
			time.Sleep(50 * time.Millisecond)
		}

		Convey("When run workers with Observ func, job duration should be great 50ms", func() {
			for i := 0; i < 5; i++ {
				worker.
					New(job).
					SetObserver(obs).
					Run(context.Background())
			}
		})
	})
}

func TestImmediately(t *testing.T) {
	schedule := func(ctx context.Context, j worker.Job) worker.Job {
		return func(ctx context.Context) {
			j(ctx)
		}
	}

	Convey("Given immediately worker with increment job", t, func() {
		var i int32
		job := func(ctx context.Context) {
			atomic.AddInt32(&i, 1)
		}

		wrk := worker.New(job).SetImmediately(true)
		Convey("It should run job 2 times", func() {
			// immediately and by schedule
			wrk.BySchedule(schedule).Run(context.Background())
			So(atomic.LoadInt32(&i), ShouldEqual, 2)
		})

		Convey("It should run job 1 imes", func() {
			wrk.Run(context.Background())
			So(atomic.LoadInt32(&i), ShouldEqual, 1)
		})
	})

	Convey("Given wait context immediately worker", t, func() {
		var i int32
		res := make(chan struct{})
		job := func(ctx context.Context) {
			<-ctx.Done()
			atomic.AddInt32(&i, 1)
			res <- struct{}{}
		}

		wrk := worker.New(job).SetImmediately(true)
		Convey("When run worker with +1 schedule", func() {
			ctx, cancel := context.WithCancel(context.Background())
			go wrk.BySchedule(schedule).Run(ctx)

			Convey("Job should executed once when context canceled", func() {
				cancel()
				So(readFromChannelWithTimeout(res), ShouldBeTrue)
				So(atomic.LoadInt32(&i), ShouldEqual, 1)
			})
		})

		Convey("When running without schedule", func() {
			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)

			Convey("Job should executed once", func() {
				cancel()
				So(readFromChannelWithTimeout(res), ShouldBeTrue)
				So(atomic.LoadInt32(&i), ShouldEqual, 1)
			})
		})
	})
}
