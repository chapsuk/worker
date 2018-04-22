package main

import (
	"context"
	"errors"
	"log"
	"sync/atomic"
	"time"

	"github.com/chapsuk/grace"
	"github.com/chapsuk/worker"
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

func createWorker(msg string) func(context.Context) {
	return func(ctx context.Context) {
		log.Printf("start %s", msg)
		time.Sleep(2 * time.Second)
		log.Printf("stop %s", msg)
	}
}

func main() {
	g := worker.NewGroup()

	w1 := worker.ByTicker(time.Second, createWorker("ticker job"))
	w2 := worker.ByTimer(time.Second, createWorker("timer job"))
	w3 := worker.WithLock(&locker{}, createWorker("with lock job"))
	w4 := worker.Many(10, w3)
	w5 := worker.ByTicker(time.Second, w4)

	g.Add(w1, w2, w3, w4, w5)
	g.Run()

	<-grace.ShutdownContext(context.Background()).Done()

	g.Stop()
}
