package worker_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestByTimer(t *testing.T) {
	Convey("Check timer, every 1s for 5s", t, func() {
		wb := &workerBuf{}
		w := worker.ByTimer(createWorkerTimingsLog(wb), time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		w(ctx)
		So(len(wb.log), ShouldBeGreaterThan, 3)
		So(wb.log, ShouldBeChronological)
	})
}

func TestByTicker(t *testing.T) {
	Convey("Check ticker schedule, every 1s for 5s", t, func() {

		wb := &workerBuf{}
		w := worker.ByTicker(createWorkerTimingsLog(wb), time.Second)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		w(ctx)
		So(len(wb.log), ShouldBeGreaterThan, 3)
		So(wb.log, ShouldBeChronological)
	})
}

func TestByCronSchedule(t *testing.T) {
	Convey("Check cron schedule, every 1s for 5s", t, func() {

		wb := &workerBuf{}
		w, err := worker.ByCronSchedule(createWorkerTimingsLog(wb), "@every 1s")
		So(err, ShouldBeNil)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		w(ctx)
		So(len(wb.log), ShouldBeGreaterThan, 3)
		So(wb.log, ShouldBeChronological)
	})
}

type workerBuf struct {
	log []time.Time
	mu  sync.Mutex
}

func createWorkerTimingsLog(wb *workerBuf) worker.Worker {
	return func(ctx context.Context) {
		wb.mu.Lock()
		wb.log = append(wb.log, time.Now())
		wb.mu.Unlock()
	}
}
