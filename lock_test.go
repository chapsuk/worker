package worker_test

import (
	"testing"
	"time"

	"github.com/chapsuk/worker"
	"github.com/go-redis/redis"
	. "github.com/smartystreets/goconvey/convey"
)

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
