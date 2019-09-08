package worker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/chapsuk/worker"
	. "github.com/smartystreets/goconvey/convey"
)

func TestWithLock(t *testing.T) {

	Convey("Given waitnig context job", t, func() {
		var (
			i     int32
			start = make(chan struct{})
			stop  = make(chan struct{})
			job   = func(ctx context.Context) {
				atomic.AddInt32(&i, 1)
				start <- struct{}{}
				<-ctx.Done()
				atomic.AddInt32(&i, 1)
				stop <- struct{}{}
			}
		)

		Convey("When run worker with custom locker", func() {
			wrk := worker.New(job).WithLock(&customLocker{})

			ctx, cancel := context.WithCancel(context.Background())
			go wrk.Run(ctx)
			So(readFromChannelWithTimeout(start), ShouldBeTrue)

			Convey("repeat run should not be execute job", func() {
				wrk.Run(ctx)
				So(atomic.LoadInt32(&i), ShouldEqual, 1)
			})

			Convey("cancel context should complete job", func() {
				cancel()
				So(readFromChannelWithTimeout(stop), ShouldBeTrue)
				So(atomic.LoadInt32(&i), ShouldEqual, 2)
			})
		})
	})
}

type customLocker struct {
	locked int32
}

func (c *customLocker) Lock() error {
	if atomic.CompareAndSwapInt32(&c.locked, 0, 1) {
		return nil
	}
	return errors.New("locked")
}

func (c *customLocker) Unlock() {
	atomic.StoreInt32(&c.locked, 0)
}
