package worker_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/dig"
)

func TestGroup(t *testing.T) {
	Convey("Given empty workers group", t, func() {
		wk := worker.NewGroup()
		So(wk, ShouldNotBeNil)

		Convey("When add 3 workers", func() {
			var (
				counter int32
				res     = make(chan struct{})
			)

			wk.Add(
				worker.New(createFakeJob(&counter, res)),
				worker.New(createFakeJob(&counter, res)),
				worker.New(createFakeJob(&counter, res)),
			)

			Convey("workers should not be started", func() {
				So(atomic.LoadInt32(&counter), ShouldEqual, 0)
			})

			Convey("When run group with 3 workers", func() {
				wk.Run()
				for i := 0; i < 3; i++ {
					So(readFromChannelWithTimeout(res), ShouldBeTrue)
				}

				Convey("all workers should be started", func() {
					So(atomic.LoadInt32(&counter), ShouldEqual, 3)
				})

				Convey("When add worker after group run", func() {
					wk.Add(worker.New(createFakeJob(&counter, res)))
					So(readFromChannelWithTimeout(res), ShouldBeTrue)

					Convey("added worker should be executed", func() {
						So(atomic.LoadInt32(&counter), ShouldEqual, 4)
					})
				})

				Convey("Stop workers call", func() {
					ch := make(chan struct{})
					go func() {
						wk.Stop()
						ch <- struct{}{}
					}()

					Convey("should not be blocking", func() {
						select {
						case <-ch:
							So("non-blocking", ShouldEqual, "non-blocking")
						case <-time.Tick(time.Second):
							So("blocking", ShouldEqual, "non-blocking")
						}
					})
				})
			})
		})
	})
}

type DigWorkerResult struct {
	dig.Out
	TestWorker *worker.Worker `group:"workers"`
}

var (
	digCounter    int32
	digResultChan chan struct{}
)

func firstDigWorker() DigWorkerResult {
	return DigWorkerResult{
		TestWorker: worker.New(createFakeJob(&digCounter, digResultChan)),
	}
}

func secondDigWorker() DigWorkerResult {
	return DigWorkerResult{
		TestWorker: worker.New(createFakeJob(&digCounter, digResultChan)),
	}
}

func TestDigGroup(t *testing.T) {
	Convey("Given dig DI Container, test worker provided", t, func() {
		dic := dig.New()
		digCounter = 0
		digResultChan = make(chan struct{})

		Convey("When provide NewDigGroup and first test dig worker, errors should be nil", func() {
			err := dic.Provide(firstDigWorker)
			So(err, ShouldBeNil)
			err = dic.Provide(worker.NewDigGroup)
			So(err, ShouldBeNil)

			Convey("Invoke workers group should return group instance", func() {
				err = dic.Invoke(func(wg *worker.Group) {
					So(wg, ShouldNotBeNil)
				})
				So(err, ShouldBeNil)
			})

			Convey("Provide one more group worker, should not return error", func() {
				err := dic.Provide(firstDigWorker)
				So(err, ShouldBeNil)

				Convey("Run worker group should start both workers", func() {
					err = dic.Invoke(func(wg *worker.Group) {
						So(wg, ShouldNotBeNil)
						wg.Run()
						So(readFromChannelWithTimeout(digResultChan), ShouldBeTrue)
						So(readFromChannelWithTimeout(digResultChan), ShouldBeTrue)
						So(atomic.LoadInt32(&digCounter), ShouldEqual, 2)
					})
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func createFakeJob(counter *int32, result chan struct{}) worker.Job {
	return func(ctx context.Context) {
		atomic.AddInt32(counter, 1)
		select {
		case result <- struct{}{}:
		case <-ctx.Done():
		}
	}
}

func readFromChannelWithTimeout(result chan struct{}) bool {
	select {
	case <-result:
		return true
	case <-time.Tick(time.Second):
		return false
	}
}
