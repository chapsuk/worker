package main

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/job"
	"github.com/chapsuk/wait"
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

func createJob(msg string) func(context.Context) {
	return func(ctx context.Context) {
		log.Printf("start %s", msg)
		time.Sleep(2 * time.Second)
		log.Printf("stop %s", msg)
	}
}

func main() {
	tickerJob := job.ByTicker(time.Second, createJob("ticker job"))
	timerJob := job.ByTicker(time.Second, createJob("timer job"))

	lockJob := job.WithLock(&locker{}, createJob("locker job"))
	tickerLockJob := job.ByTicker(time.Second, lockJob)

	withBefore := job.Before(tickerLockJob, createJob("before job"))
	withAfter := job.After(tickerLockJob, createJob("after job"))

	ctx := grace.ShutdownContext(context.Background())

	wg := wait.Group{}

	wg.AddWithContext(ctx, tickerJob)
	wg.AddWithContext(ctx, timerJob)
	wg.AddWithContext(ctx, lockJob)
	wg.AddWithContext(ctx, tickerLockJob)
	wg.AddWithContext(ctx, withBefore)
	wg.AddWithContext(ctx, withAfter)

	wg.Wait()
}
