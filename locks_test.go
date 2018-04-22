package worker_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	"github.com/go-redis/redis"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"
)

type locker struct {
	locked uint32
}

func (l *locker) Lock() error {
	if atomic.CompareAndSwapUint32(&l.locked, 0, 1) {
		return nil
	}
	return errors.New("locked")
}

func (l *locker) Unlock() {
	atomic.StoreUint32(&l.locked, 0)
}

func TestSimpleLocker(t *testing.T) {

	Convey("Given group of 3 workers with simple locker", t, func() {
		lock := &locker{}

		var counter int32

		wk := worker.NewGroup()

		wk.Add(
			worker.WithLock(createFakeWorkerWithTimeout(&counter, 2*time.Second), lock),
			worker.WithLock(createFakeWorkerWithTimeout(&counter, 2*time.Second), lock),
			worker.WithLock(createFakeWorkerWithTimeout(&counter, 2*time.Second), lock),
		)

		Convey("When run workers group", func() {
			wk.Run()

			Convey("worker should executed once", func() {
				wk.Stop()
				So(atomic.LoadInt32(&counter), ShouldEqual, 1)
			})
		})
	})
}

func TestRedisLocks(t *testing.T) {

	Convey("Given workers group with redis locker usage", t, func() {
		var counter int32

		opts := worker.RedisLockOptions{
			LockKey:  "testRedisLocks",
			LockTTL:  5 * time.Second,
			RedisCLI: getRedisClient(),
			Logger:   zap.S(),
		}
		wk := worker.NewGroup()
		wk.Add(
			worker.WithRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
		)

		Convey("When run...", func() {
			wk.Run()
			time.Sleep(time.Second)

			Convey("only one worker should be running", func() {
				wk.Stop()
				So(atomic.LoadInt32(&counter), ShouldEqual, 1)
			})
		})
	})

	Convey("Given bsm redis locker options, retry setting guarantee each worker will be runned", t, func() {
		var counter int32

		opts := worker.BsmRedisLockOptions{
			RedisLockOptions: worker.RedisLockOptions{
				LockKey:  "testBsmRedisLockRetries",
				LockTTL:  5 * time.Second,
				RedisCLI: getRedisClient(),
				Logger:   zap.S(),
			},
			RetryCount: 100,
			RetryDelay: 50 * time.Millisecond,
		}

		wk := worker.NewGroup()
		wk.Add(
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, 100*time.Millisecond), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, 100*time.Millisecond), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, 100*time.Millisecond), opts),
		)

		Convey("Run group should has 3 completed jobs, retries in flight", func() {
			wk.Run()
			time.Sleep(time.Second)

			wk.Stop()
			So(atomic.LoadInt32(&counter), ShouldEqual, 3)
		})
	})

	Convey("Given bsm locker, check without retris", t, func() {
		var counter int32

		opts := worker.BsmRedisLockOptions{
			RedisLockOptions: worker.RedisLockOptions{
				LockKey:  "testBsmRedisLock",
				LockTTL:  5 * time.Second,
				RedisCLI: getRedisClient(),
			},
			RetryCount: 0,
			RetryDelay: 0,
		}

		wk := worker.NewGroup()
		wk.Add(
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
		)

		Convey("only one worker should be completed", func() {
			wk.Run()
			wk.Stop()
			So(atomic.LoadInt32(&counter), ShouldEqual, 1)
		})
	})

	Convey("When incorrect redis connection", t, func() {
		var counter int32

		opts := worker.BsmRedisLockOptions{
			RedisLockOptions: worker.RedisLockOptions{
				LockKey:  "testMissingConnection",
				LockTTL:  5 * time.Second,
				RedisCLI: redis.NewClient(&redis.Options{Addr: "foo.bar"}),
			},
			RetryCount: 0,
			RetryDelay: 0,
		}

		wk := worker.NewGroup()
		wk.Add(
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
			worker.WithBsmRedisLock(
				createFakeWorkerWithTimeout(&counter, time.Second), opts),
		)

		Convey("no one worker should be running", func() {
			wk.Run()
			wk.Stop()
			So(atomic.LoadInt32(&counter), ShouldEqual, 0)
		})
	})
}

func getRedisClient() *redis.Client {
	cli := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})
	So(cli.Ping().Err(), ShouldBeNil)
	return cli
}

func createFakeWorkerWithTimeout(counter *int32, timeout time.Duration) worker.Worker {
	return func(ctx context.Context) {
		atomic.AddInt32(counter, 1)
		time.Sleep(timeout)
	}
}
