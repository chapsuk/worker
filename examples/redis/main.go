// !require redis on localhost:6379
// docker run -p6379:6379 -d redis
package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
	"github.com/go-redis/redis"
	"go.uber.org/zap"
)

func main() {
	redis := getRedisClient()

	ropts1 := createRedisLockOpts(redis, "lock1", time.Hour)
	ropts2 := createRedisLockOpts(redis, "lock2", time.Hour)

	bopts3 := createBsmLockOpts(redis, "lock3", time.Hour, time.Second, 0)
	bopts4 := createBsmLockOpts(redis, "lock4", time.Hour, time.Millisecond, 10)

	var r1, r2, r3, r4 int32

	job1 := createIncrementJobWithTimeout("job1", &r1, 1, time.Second)
	job2 := createIncrementJobWithTimeout("job2", &r2, 1, time.Second)
	job3 := createIncrementJobWithTimeout("job3", &r3, 1, time.Second)
	job4 := createIncrementJobWithTimeout("job4", &r4, 1, time.Second)

	wk := worker.NewGroup()

	for i := 0; i < 20; i++ {
		wk.Add(
			worker.New(job1).
				WithRedisLock(ropts1).
				ByTicker(time.Second),

			worker.New(job2).
				WithRedisLock(ropts2).
				ByTicker(time.Second),

			worker.New(job3).
				WithBsmRedisLock(bopts3).
				ByTicker(time.Second),

			worker.New(job4).
				WithBsmRedisLock(bopts4).
				ByTicker(time.Second),
		)
	}

	log.Print("starting workers...")
	wk.Run()

	<-grace.ShutdownContext(context.Background()).Done()

	wk.Stop()
	log.Print("stopped")
}

func createIncrementJobWithTimeout(name string, target *int32, delta int32, timeout time.Duration) func(context.Context) {
	return func(ctx context.Context) {
		log.Printf("%s start, int before: %d after %d", name, atomic.LoadInt32(target), atomic.AddInt32(target, delta))
		time.Sleep(timeout)
	}
}

func getRedisClient() *redis.Client {
	log.Print("create redis client")
	cli := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := cli.Ping().Err(); err != nil {
		panic("redis ping error: " + err.Error())
	}
	return cli
}

func createRedisLockOpts(cli *redis.Client, lockkey string, lockttl time.Duration) worker.RedisLockOptions {
	return worker.RedisLockOptions{
		LockKey:  lockkey,
		LockTTL:  lockttl,
		RedisCLI: cli,
		Logger:   zap.S(),
	}
}

func createBsmLockOpts(
	cli *redis.Client,
	lockkey string,
	lockttl, retryDelay time.Duration,
	retries int,
) worker.BsmRedisLockOptions {
	return worker.BsmRedisLockOptions{
		RedisLockOptions: createRedisLockOpts(cli, lockkey, lockttl),
		RetryCount:       retries,
		RetryDelay:       retryDelay,
	}
}
