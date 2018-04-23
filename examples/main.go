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
	log.Print("create redis client")
	rcli := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	if err := rcli.Ping().Err(); err != nil {
		panic("redis ping error: " + err.Error())
	}

	log.Print("create redis lock options")
	ropts := worker.RedisLockOptions{
		LockKey:  "lock_key",
		LockTTL:  5 * time.Second,
		RedisCLI: rcli,
		Logger:   zap.S(),
	}

	log.Print("create redis lock options for lock by bsm/redis-lock pkg")
	opts := worker.BsmRedisLockOptions{
		RedisLockOptions: ropts,
		RetryCount:       3,
		RetryDelay:       2,
	}

	log.Print("define job sample")
	var i int32
	job := func(ctx context.Context) {
		atomic.AddInt32(&i, 1)
		log.Printf("run #%d", atomic.LoadInt32(&i))
		<-ctx.Done()
	}

	log.Print("define jobs with redis locks")
	job1 := worker.WithBsmRedisLock(job, opts)
	job2 := worker.WithRedisLock(job, ropts)

	log.Print("create job with cron format schedule")
	w1, err := worker.ByCronSchedule(job1, "@every 1s")
	if err != nil {
		panic("parse cron spec error: " + err.Error())
	}

	log.Print("create jobs with ticker schedule")
	w2 := worker.ByTicker(job1, time.Second)
	w3 := worker.ByTicker(job2, time.Second)

	log.Print("create jobs with timer schedule")
	w4 := worker.ByTimer(job1, time.Second)
	w5 := worker.ByTimer(job2, time.Second)

	log.Print("create workers group, init and run")
	wk := worker.NewGroup()
	wk.Add(w1, w2, w3, w4, w5)
	wk.Run()

	log.Print("wait stop signal")
	<-grace.ShutdownContext(context.Background()).Done()

	log.Print("stopping")
	wk.Stop()
	log.Print("gracefully stopped")
}
