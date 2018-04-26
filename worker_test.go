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
				worker.New(job).WithMetrics(obs).Run(context.Background())
			}
		})
	})
}

func TestImmediately(t *testing.T) {
	Convey("Given worker with increment job", t, func() {
		var i int32
		job := func(ctx context.Context) {
			atomic.AddInt32(&i, 1)
		}

		wrk := worker.New(job)
		Convey("It should run job 2 times", func() {
			// immediatly and by schedule
			wrk.BySchedule(func(ctx context.Context, j worker.Job) worker.Job {
				return func(ctx context.Context) {
					j(ctx)
				}
			}).SetImmediately(true).Run(context.Background())
			time.Sleep(time.Millisecond)
			So(atomic.LoadInt32(&i), ShouldEqual, 2)
		})
	})
}
