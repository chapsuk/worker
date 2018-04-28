package worker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chapsuk/worker"
	"github.com/go-redis/redis"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRedisLock(t *testing.T) {

	Convey("Given redis client", t, func() {
		redis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

		if err := redis.Ping().Err(); err != nil {
			SkipConvey("redis connection error: "+err.Error(), func() {})
			return
		}

		Convey("Create redis lock options and job who wait context", func() {
			lgr := &logger{}
			opts := worker.RedisLockOptions{
				RedisCLI: redis,
				LockKey:  fmt.Sprintf("redis_lock_test%d", time.Now().Nanosecond()),
				LockTTL:  time.Second,
				Logger:   lgr,
			}

			var (
				i     int32
				start = make(chan struct{})
				stop  = make(chan struct{})
			)

			job := func(ctx context.Context) {
				start <- struct{}{}
				atomic.AddInt32(&i, 1)
				<-ctx.Done()
				atomic.AddInt32(&i, 1)
				stop <- struct{}{}
			}

			Convey("When run job with redis lock", func() {
				wrk := worker.New(job).WithRedisLock(opts)

				ctx, cancel := context.WithCancel(context.Background())
				go wrk.Run(ctx)
				So(readFromChannelWithTimeout(start), ShouldBeTrue)

				Convey("Repeat run should not be executed", func() {
					wrk.Run(ctx)
					So(atomic.LoadInt32(&i), ShouldEqual, 1)
					So(lgr.warns(), ShouldEqual, 1)

					Convey("After lock expired new job should start", func() {
						time.Sleep(2 * time.Second)
						go wrk.Run(ctx)
						So(readFromChannelWithTimeout(start), ShouldBeTrue)
						So(atomic.LoadInt32(&i), ShouldEqual, 2)
						So(lgr.warns(), ShouldEqual, 1)

						Convey("Cancel context should stop all runeed jobs", func() {
							cancel()
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)

							So(atomic.LoadInt32(&i), ShouldEqual, 4)
							So(lgr.warns(), ShouldEqual, 1)
						})
					})
				})
			})

			Convey("Run job with lock", func() {
				wrk := worker.New(job).WithRedisLock(opts)

				ctx, cancel := context.WithCancel(context.Background())
				go wrk.Run(ctx)
				So(readFromChannelWithTimeout(start), ShouldBeTrue)

				Convey("Release lock should logging error if redis shutdown", func() {
					redis.Close()
					cancel()
					So(readFromChannelWithTimeout(stop), ShouldBeTrue)
					<-time.Tick(time.Second)
					So(lgr.errs(), ShouldEqual, 1)
				})
			})
		})
	})

	Convey("Given not accessible redis client", t, func() {
		redis := redis.NewClient(&redis.Options{
			Addr:        "127.0.0.2:8080",
			DialTimeout: time.Millisecond,
		})

		lgr := &logger{}
		opts := worker.RedisLockOptions{
			RedisCLI: redis,
			LockKey:  fmt.Sprintf("rlk%d", time.Now().Nanosecond()),
			LockTTL:  time.Second,
			Logger:   lgr,
		}

		Convey("When try run worker", func() {
			var i int32
			worker.
				New(func(context.Context) { i++ }).
				WithRedisLock(opts).
				Run(context.TODO())

			Convey("Looger should get error message", func() {
				So(lgr.errs(), ShouldEqual, 1)
			})
		})
	})
}

func TestBsmRedisLock(t *testing.T) {

	Convey("Given redis client", t, func() {
		redis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

		if err := redis.Ping().Err(); err != nil {
			SkipConvey("redis connection error: "+err.Error(), func() {})
			return
		}

		Convey("Create bsm redis lock options and job who wait context", func() {
			opts := worker.BsmRedisLockOptions{
				RedisLockOptions: worker.RedisLockOptions{
					RedisCLI: redis,
					LockKey:  fmt.Sprintf("bsm_redis_lock_test%d", time.Now().Nanosecond()),
					LockTTL:  time.Second,
				},
				RetryCount: 2,
				RetryDelay: 550 * time.Millisecond,
			}

			var (
				i     int32
				start = make(chan struct{})
				stop  = make(chan struct{})
			)

			job := func(ctx context.Context) {
				atomic.AddInt32(&i, 1)
				start <- struct{}{}
				<-ctx.Done()
				atomic.AddInt32(&i, 1)
				stop <- struct{}{}
			}

			Convey("When run job with bsm redis lock", func() {
				wrk := worker.New(job).WithBsmRedisLock(opts)

				ctx, cancel := context.WithCancel(context.Background())

				go wrk.Run(ctx)

				select {
				case <-start:
					So(atomic.LoadInt32(&i), ShouldEqual, 1)
				case <-time.Tick(2 * time.Second):
					So("job not started", ShouldBeFalse)
				}

				Convey("Repeat run should be executed by retries", func() {
					go wrk.Run(ctx)

					select {
					case <-start:
						So(atomic.LoadInt32(&i), ShouldEqual, 2)
					case <-time.Tick(2 * time.Second):
						So("job not started", ShouldBeFalse)
					}

					Convey("After lock expired new job should start", func() {
						time.Sleep(2 * time.Second)
						go wrk.WithBsmRedisLock(opts.NewWith(
							opts.LockKey, opts.LockTTL, 0, time.Second,
						)).Run(ctx)

						select {
						case <-start:
							So(atomic.LoadInt32(&i), ShouldEqual, 3)
						case <-time.Tick(2 * time.Second):
							So("job not started", ShouldBeFalse)
						}

						Convey("Cancel context should stop all runeed jobs", func() {
							cancel()
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(atomic.LoadInt32(&i), ShouldEqual, 6)
						})
					})
				})
			})
		})

		Convey("Create bsm redis lock options without retries and job who wait context", func() {
			lgr := &logger{}
			opts := worker.BsmRedisLockOptions{
				RedisLockOptions: worker.RedisLockOptions{
					RedisCLI: redis,
					LockKey:  fmt.Sprintf("bsm_redis_lock_test%d", time.Now().Nanosecond()),
					LockTTL:  time.Second,
					Logger:   lgr,
				},
				RetryCount: 0,
				// RetryDelay: 550 * time.Millisecond,
			}

			var (
				i     int32
				start = make(chan struct{})
				stop  = make(chan struct{})
			)

			job := func(ctx context.Context) {
				atomic.AddInt32(&i, 1)
				start <- struct{}{}
				<-ctx.Done()
				atomic.AddInt32(&i, 1)
				stop <- struct{}{}
			}

			Convey("When run job with bsm redis lock", func() {
				wrk := worker.New(job).WithBsmRedisLock(opts)

				ctx, cancel := context.WithCancel(context.Background())
				go wrk.Run(ctx)
				select {
				case <-start:
					So(atomic.LoadInt32(&i), ShouldEqual, 1)
				case <-time.Tick(3 * time.Second):
					So("run worker, to slow", ShouldBeFalse)
				}

				Convey("Repeat run should not execute by retries", func() {
					wrk.Run(ctx)
					So(atomic.LoadInt32(&i), ShouldEqual, 1)
					So(lgr.warns(), ShouldEqual, 1)

					Convey("After lock expired new job should start", func() {
						time.Sleep(2 * time.Second)
						go wrk.Run(ctx)

						select {
						case <-start:
							So(atomic.LoadInt32(&i), ShouldEqual, 2)
							So(lgr.warns(), ShouldEqual, 1)
						case <-time.Tick(3 * time.Second):
							So("run worker, to slow", ShouldBeFalse)
						}

						Convey("Cancel context should stop all runeed jobs", func() {
							cancel()
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(readFromChannelWithTimeout(stop), ShouldBeTrue)
							So(atomic.LoadInt32(&i), ShouldEqual, 4)
							// first job lock expired, logger should get warn
							<-time.Tick(time.Second)
							So(lgr.warns(), ShouldEqual, 2)
						})
					})
				})
			})
		})
	})

	Convey("Given not accessible redis client", t, func() {
		redis := redis.NewClient(&redis.Options{
			Addr:        "127.0.0.2:8080",
			DialTimeout: time.Millisecond,
		})

		lgr := &logger{}
		opts := worker.BsmRedisLockOptions{
			RedisLockOptions: worker.RedisLockOptions{
				RedisCLI: redis,
				LockKey:  fmt.Sprintf("bsm%d", time.Now().Nanosecond()),
				LockTTL:  time.Second,
				Logger:   lgr,
			},
		}

		Convey("When try run worker", func() {
			worker.
				New(func(context.Context) {}).
				WithBsmRedisLock(opts).
				Run(context.TODO())

			Convey("Looger should get 1 error message", func() {
				So(lgr.errs(), ShouldEqual, 1)
			})
		})
	})
}

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

func TestRedisOptions(t *testing.T) {

	Convey("Given redis lock options", t, func() {
		redis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

		sopts := worker.RedisLockOptions{
			RedisCLI: redis,
			LockKey:  "gen",
			LockTTL:  time.Second,
		}

		Convey("When create new options from source options", func() {
			dopts := sopts.NewWith("new", 2*time.Second)

			Convey("original options should not be changed", func() {
				So(sopts.LockKey, ShouldEqual, "gen")
				So(sopts.LockTTL, ShouldEqual, time.Second)
				So(sopts.RedisCLI, ShouldPointTo, redis)
			})

			Convey("new options should be filling", func() {
				So(dopts.LockKey, ShouldEqual, "new")
				So(dopts.LockTTL, ShouldEqual, 2*time.Second)
				So(dopts.RedisCLI, ShouldPointTo, redis)
			})
		})
	})
}

func TestBsmRedisOptions(t *testing.T) {

	Convey("Given bsm lock options", t, func() {
		redis := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})

		sopts := worker.BsmRedisLockOptions{
			RedisLockOptions: worker.RedisLockOptions{
				RedisCLI: redis,
				LockKey:  "gen",
				LockTTL:  time.Second,
			},
			RetryCount: 1,
			RetryDelay: time.Second,
		}

		Convey("When create new options from source options", func() {
			dopts := sopts.NewWith("new", 2*time.Second, 2, 3*time.Second)

			Convey("original options should not be changed", func() {
				So(sopts.LockKey, ShouldEqual, "gen")
				So(sopts.LockTTL, ShouldEqual, time.Second)
				So(sopts.RetryCount, ShouldEqual, 1)
				So(sopts.RetryDelay, ShouldEqual, time.Second)
				So(sopts.RedisCLI, ShouldPointTo, redis)
			})

			Convey("new options should be filling", func() {
				So(dopts.LockKey, ShouldEqual, "new")
				So(dopts.LockTTL, ShouldEqual, 2*time.Second)
				So(dopts.RetryCount, ShouldEqual, 2)
				So(dopts.RetryDelay, ShouldEqual, 3*time.Second)
				So(dopts.RedisCLI, ShouldPointTo, redis)
			})
		})
	})
}

type logger struct {
	wrnw []string
	errw []string
	mu   sync.RWMutex
}

func (l *logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.mu.Lock()
	l.errw = append(l.errw, msg)
	l.mu.Unlock()
}

func (l *logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.mu.Lock()
	l.wrnw = append(l.wrnw, fmt.Sprintf("msg: %s kv: %+v", msg, keysAndValues))
	l.mu.Unlock()
}

func (l *logger) warns() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.wrnw)
}

func (l *logger) errs() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.errw)
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
