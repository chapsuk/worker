package worker_test

import (
	"context"
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
